package handler

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"backend/internal/domain"
	"github.com/go-pdf/fpdf"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
)

const (
	pdfFont       = "Golos"
	pdfPageWidth  = 210.0
	pdfPageMargin = 17.0
)

type offerPDFLine struct {
	Name        string
	Description string
	Price       int64
}

type offerPDFModel struct {
	DocumentDate string
	Delivery     string
	Finishing    string
	Lines        []offerPDFLine
	Discount     int64
	Conditions   []string
}

func buildOfferPDF(document *domain.OfferDocument) ([]byte, error) {
	if document == nil || document.Offer == nil {
		return nil, errors.New("offer document is required")
	}
	model := newOfferPDFModel(document)
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(pdfPageMargin, 18, pdfPageMargin)
	pdf.SetAutoPageBreak(true, 18)
	pdf.AddUTF8FontFromBytes(pdfFont, "", goregular.TTF)
	pdf.AddUTF8FontFromBytes(pdfFont, "B", gobold.TTF)
	configureOfferPDFChrome(pdf, document)
	pdf.AddPage()

	drawOfferHero(pdf, document, model)
	drawApartmentSummary(pdf, document, model)
	drawApartmentPlan(pdf, document)
	drawPriceComposition(pdf, document, model)
	drawOfferConditions(pdf, document, model)
	drawOfferContacts(pdf, document)

	var output bytes.Buffer
	if err := pdf.Output(&output); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func newOfferPDFModel(document *domain.OfferDocument) offerPDFModel {
	offer := document.Offer
	createdAt := offer.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	apartmentPrice := offer.BasePrice - offer.ParkingPrice - offer.StoragePrice
	lines := []offerPDFLine{{
		Name:        "Квартира №" + document.ApartmentNumber,
		Description: fmt.Sprintf("%d-комнатная · %s м² · %d этаж", document.ApartmentRooms, formatArea(document.ApartmentArea), document.ApartmentFloor),
		Price:       apartmentPrice,
	}}
	if offer.ParkingUnitID != nil {
		lines = append(lines, offerPDFLine{
			Name:        "Машино-место " + optionalUnitNumber(offer.ParkingNumber, offer.ParkingUnitID),
			Description: optionalArea(document.ParkingArea),
			Price:       offer.ParkingPrice,
		})
	}
	if offer.StorageUnitID != nil {
		lines = append(lines, offerPDFLine{
			Name:        "Кладовая " + optionalUnitNumber(offer.StorageNumber, offer.StorageUnitID),
			Description: optionalArea(document.StorageArea),
			Price:       offer.StoragePrice,
		})
	}
	delivery := "Срок передачи определяется договором долевого участия"
	if document.ForecastDate != nil && valueOrZero(document.DeliveryShiftDays) <= 0 {
		delivery = "Прогноз передачи: " + russianDate(*document.ForecastDate)
	} else if document.PlannedDate != nil && valueOrZero(document.DeliveryShiftDays) <= 0 {
		delivery = "Плановый срок: " + russianDate(*document.PlannedDate)
	}
	return offerPDFModel{
		DocumentDate: russianDate(createdAt),
		Delivery:     delivery,
		Finishing:    finishingLabel(document.ApartmentFinishing),
		Lines:        lines,
		Discount:     offer.BasePrice - offer.FinalPrice,
		Conditions: []string{
			"Стоимость и доступность указанных объектов актуальны на дату формирования КП и окончательно фиксируются договором.",
			"Срок передачи квартиры, характеристики отделки и комплект документов определяются условиями ДДУ и проектной документацией.",
			"Схема отражает функциональное зонирование квартиры. Точные размеры, расположение инженерных коммуникаций и оборудования уточняются по рабочей документации.",
			"Настоящий документ носит информационный характер и не является публичной офертой.",
		},
	}
}

func configureOfferPDFChrome(pdf *fpdf.Fpdf, document *domain.OfferDocument) {
	pdf.SetTitle(fmt.Sprintf("Коммерческое предложение ДСК №%d", document.Offer.ID), true)
	pdf.SetAuthor("АО СЗ «ДСК»", true)
	pdf.SetCreator("AI-помощник отдела продаж ДСК", true)
	pdf.SetHeaderFuncMode(func() {
		pdf.SetFillColor(8, 55, 91)
		pdf.Rect(0, 0, pdfPageWidth, 10, "F")
	}, true)
	pdf.SetFooterFunc(func() {
		pdf.SetY(-12)
		pdf.SetDrawColor(221, 227, 231)
		pdf.Line(pdfPageMargin, pdf.GetY(), pdfPageWidth-pdfPageMargin, pdf.GetY())
		pdf.SetY(-9)
		pdf.SetFont(pdfFont, "", 7.5)
		pdf.SetTextColor(95, 111, 122)
		pdf.CellFormat(120, 5, "АО СЗ «ДСК» · Коммерческое предложение", "", 0, "L", false, 0, "")
		pdf.CellFormat(54, 5, fmt.Sprintf("Страница %d", pdf.PageNo()), "", 0, "R", false, 0, "")
	})
}

func drawOfferHero(pdf *fpdf.Fpdf, document *domain.OfferDocument, model offerPDFModel) {
	setPDFColor(pdf, "ink")
	drawDSKMark(pdf, pdfPageMargin, 18)
	pdf.SetXY(51, 19)
	pdf.SetFont(pdfFont, "B", 9)
	pdf.SetTextColor(8, 55, 91)
	pdf.Cell(55, 5, "АО СЗ «ДСК»")
	pdf.SetXY(51, 24)
	pdf.SetFont(pdfFont, "", 7.5)
	pdf.SetTextColor(95, 111, 122)
	pdf.Cell(70, 4, "Отдел продаж жилой недвижимости")
	pdf.SetXY(138, 19)
	pdf.SetFont(pdfFont, "", 8)
	pdf.CellFormat(55, 5, fmt.Sprintf("КП №%d · версия %d", document.Offer.ID, document.Offer.Version), "", 1, "R", false, 0, "")
	pdf.SetX(138)
	pdf.CellFormat(55, 5, model.DocumentDate, "", 1, "R", false, 0, "")

	pdf.SetY(40)
	pdf.SetFont(pdfFont, "B", 24)
	pdf.SetTextColor(12, 31, 43)
	pdf.Cell(0, 11, "Коммерческое предложение")
	pdf.Ln(13)
	pdf.SetFont(pdfFont, "", 11)
	pdf.SetTextColor(62, 83, 96)
	recipient := strings.Join(nonEmpty("Клиент: "+document.ClientName, document.ClientEmail, document.ComplexName), " · ")
	pdf.MultiCell(0, 6, recipient, "", "L", false)
	pdf.Ln(5)

	pdf.SetFillColor(239, 246, 250)
	pdf.SetDrawColor(207, 221, 230)
	y := pdf.GetY()
	pdf.RoundedRect(pdfPageMargin, y, 176, 25, 3, "1234", "DF")
	pdf.SetXY(pdfPageMargin+6, y+5)
	pdf.SetFont(pdfFont, "B", 10)
	pdf.SetTextColor(8, 55, 91)
	pdf.Cell(0, 5, document.ComplexName)
	pdf.SetXY(pdfPageMargin+6, y+11)
	pdf.SetFont(pdfFont, "", 8.5)
	pdf.SetTextColor(62, 83, 96)
	pdf.Cell(0, 5, strings.Join(nonEmpty(document.BuildingAddress, document.ComplexAddress, document.BuildingDistrict), " · "))
	pdf.SetXY(pdfPageMargin+6, y+17)
	pdf.Cell(0, 5, model.Delivery)
	pdf.SetY(y + 31)
}

func drawApartmentSummary(pdf *fpdf.Fpdf, document *domain.OfferDocument, model offerPDFModel) {
	sectionTitle(pdf, "Выбранная квартира")
	y := pdf.GetY()
	cards := []struct{ label, value string }{
		{"Квартира", "№" + document.ApartmentNumber},
		{"Комнат", strconv.Itoa(document.ApartmentRooms)},
		{"Площадь", formatArea(document.ApartmentArea) + " м²"},
		{"Этаж", strconv.Itoa(document.ApartmentFloor)},
		{"Отделка", model.Finishing},
		{"Готовность", readinessLabel(document.ReadinessPercent)},
	}
	cardWidth := 56.0
	for index, card := range cards {
		row, column := index/3, index%3
		x := pdfPageMargin + float64(column)*(cardWidth+4)
		cy := y + float64(row)*18
		pdf.SetFillColor(249, 248, 244)
		pdf.SetDrawColor(226, 225, 218)
		pdf.RoundedRect(x, cy, cardWidth, 14, 2, "1234", "DF")
		pdf.SetXY(x+4, cy+2)
		pdf.SetFont(pdfFont, "", 7)
		pdf.SetTextColor(95, 111, 122)
		pdf.Cell(cardWidth-8, 4, card.label)
		pdf.SetXY(x+4, cy+7)
		pdf.SetFont(pdfFont, "B", 9)
		pdf.SetTextColor(12, 31, 43)
		pdf.Cell(cardWidth-8, 4, card.value)
	}
	pdf.SetY(y + 42)
}

func drawApartmentPlan(pdf *fpdf.Fpdf, document *domain.OfferDocument) {
	sectionTitle(pdf, "Планировка квартиры")
	x, y, width, height := pdfPageMargin, pdf.GetY(), 176.0, 82.0
	pdf.SetFillColor(252, 251, 247)
	pdf.SetDrawColor(191, 202, 209)
	pdf.SetLineWidth(0.55)
	pdf.RoundedRect(x, y, width, height, 2, "1234", "DF")
	drawFunctionalPlan(pdf, x+6, y+6, width-12, height-16, document.ApartmentRooms, document.ApartmentID)
	pdf.SetXY(x+4, y+height-8)
	pdf.SetFont(pdfFont, "", 6.5)
	pdf.SetTextColor(95, 111, 122)
	pdf.CellFormat(width-8, 4, "Схема функционального зонирования · без масштаба и обмерных размеров", "", 0, "R", false, 0, "")
	pdf.SetY(y + height + 7)
}

func drawFunctionalPlan(pdf *fpdf.Fpdf, x, y, width, height float64, rooms, apartmentID int) {
	if rooms < 1 {
		rooms = 1
	}
	if rooms > 4 {
		rooms = 4
	}
	variant := apartmentID % 3
	serviceHeight := height * 0.34
	livingHeight := height - serviceHeight
	if variant == 1 {
		serviceHeight = height * 0.38
	}
	roomWidth := width / float64(rooms)
	for index := 0; index < rooms; index++ {
		label := "Жилая комната"
		if rooms == 1 {
			label = "Комната"
		}
		drawPlanRoom(pdf, x+float64(index)*roomWidth, y, roomWidth, livingHeight, label, 232, 244, 238)
	}
	serviceY := y + livingHeight
	kitchenWidth := width * 0.4
	bathWidth := width * 0.23
	drawPlanRoom(pdf, x, serviceY, kitchenWidth, serviceHeight, "Кухня", 255, 239, 215)
	drawPlanRoom(pdf, x+kitchenWidth, serviceY, bathWidth, serviceHeight, "Санузел", 225, 239, 246)
	drawPlanRoom(pdf, x+kitchenWidth+bathWidth, serviceY, width-kitchenWidth-bathWidth, serviceHeight, "Прихожая", 240, 237, 230)

	pdf.SetDrawColor(76, 114, 137)
	pdf.SetLineWidth(0.8)
	for index := 0; index < rooms; index++ {
		wx := x + float64(index)*roomWidth + roomWidth*0.27
		pdf.Line(wx, y, wx+roomWidth*0.46, y)
		pdf.Line(wx, y+1.2, wx+roomWidth*0.46, y+1.2)
	}
	entranceX := x + width - 17 - float64(variant*3)
	pdf.SetDrawColor(244, 157, 35)
	pdf.SetLineWidth(1.2)
	pdf.Line(entranceX, y+height, entranceX+10, y+height)
	pdf.SetLineWidth(0.35)
}

func drawPlanRoom(pdf *fpdf.Fpdf, x, y, width, height float64, label string, red, green, blue int) {
	pdf.SetFillColor(red, green, blue)
	pdf.SetDrawColor(87, 110, 123)
	pdf.Rect(x, y, width, height, "DF")
	pdf.SetXY(x+1, y+height/2-2.5)
	pdf.SetFont(pdfFont, "", 7)
	pdf.SetTextColor(38, 64, 78)
	pdf.CellFormat(width-2, 5, label, "", 0, "C", false, 0, "")
}

func drawPriceComposition(pdf *fpdf.Fpdf, document *domain.OfferDocument, model offerPDFModel) {
	if pdf.GetY() > 180 {
		pdf.AddPage()
	}
	sectionTitle(pdf, "Состав предложения")
	pdf.SetFillColor(8, 55, 91)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont(pdfFont, "B", 8)
	pdf.CellFormat(112, 8, "Объект", "", 0, "L", true, 0, "")
	pdf.CellFormat(64, 8, "Стоимость", "", 1, "R", true, 0, "")
	for index, line := range model.Lines {
		if index%2 == 0 {
			pdf.SetFillColor(247, 249, 250)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		pdf.SetTextColor(12, 31, 43)
		pdf.SetFont(pdfFont, "B", 8.5)
		y := pdf.GetY()
		pdf.CellFormat(112, 12, line.Name, "", 0, "L", true, 0, "")
		pdf.SetXY(pdfPageMargin+2, y+6)
		pdf.SetFont(pdfFont, "", 6.8)
		pdf.SetTextColor(95, 111, 122)
		pdf.Cell(105, 4, line.Description)
		pdf.SetXY(pdfPageMargin+112, y)
		pdf.SetFont(pdfFont, "B", 8.5)
		pdf.SetTextColor(12, 31, 43)
		pdf.CellFormat(64, 12, formatMoney(line.Price), "", 1, "R", true, 0, "")
	}
	pdf.Ln(2)
	priceRow(pdf, "Стоимость комплекта", formatMoney(document.Offer.BasePrice), false)
	priceRow(pdf, "Скидка · "+document.Offer.DiscountPercent+"%", "− "+formatMoney(model.Discount), false)
	pdf.SetFillColor(245, 166, 35)
	pdf.SetTextColor(12, 31, 43)
	pdf.SetFont(pdfFont, "B", 12)
	pdf.CellFormat(88, 13, "Итоговая стоимость", "", 0, "L", true, 0, "")
	pdf.CellFormat(88, 13, formatMoney(document.Offer.FinalPrice), "", 1, "R", true, 0, "")
	pdf.Ln(7)
}

func priceRow(pdf *fpdf.Fpdf, label, value string, fill bool) {
	pdf.SetFont(pdfFont, "", 8.5)
	pdf.SetTextColor(62, 83, 96)
	pdf.CellFormat(88, 7, label, "", 0, "L", fill, 0, "")
	pdf.SetFont(pdfFont, "B", 8.5)
	pdf.SetTextColor(12, 31, 43)
	pdf.CellFormat(88, 7, value, "", 1, "R", fill, 0, "")
}

func drawOfferConditions(pdf *fpdf.Fpdf, document *domain.OfferDocument, model offerPDFModel) {
	sectionTitle(pdf, "Персональные условия")
	pdf.SetFont(pdfFont, "", 9)
	pdf.SetTextColor(38, 57, 68)
	personalText := normalizeOfferText(document.Offer.GeneratedText)
	if personalText == "" {
		personalText = "Состав предложения подготовлен менеджером с учётом выбранной квартиры и дополнительных позиций."
	}
	pdf.MultiCell(0, 5.2, personalText, "", "L", false)
	pdf.Ln(5)

	sectionTitle(pdf, "Условия и важная информация")
	for _, condition := range model.Conditions {
		pdf.SetFont(pdfFont, "B", 9)
		pdf.SetTextColor(245, 157, 35)
		pdf.Cell(6, 5, "•")
		pdf.SetFont(pdfFont, "", 8)
		pdf.SetTextColor(62, 83, 96)
		x, y := pdf.GetX(), pdf.GetY()
		pdf.MultiCell(170, 4.7, condition, "", "L", false)
		if pdf.GetY() < y+5 {
			pdf.SetY(y + 5)
		}
		pdf.SetX(x - 6)
		pdf.Ln(1)
	}
	pdf.Ln(3)
}

func drawOfferContacts(pdf *fpdf.Fpdf, document *domain.OfferDocument) {
	if pdf.GetY() > 248 {
		pdf.AddPage()
	}
	y := pdf.GetY()
	pdf.SetFillColor(8, 55, 91)
	pdf.RoundedRect(pdfPageMargin, y, 176, 24, 3, "1234", "F")
	pdf.SetXY(pdfPageMargin+6, y+5)
	pdf.SetFont(pdfFont, "B", 10)
	pdf.SetTextColor(255, 255, 255)
	pdf.Cell(95, 5, "Ваш менеджер · "+document.ManagerName)
	pdf.SetXY(pdfPageMargin+6, y+12)
	pdf.SetFont(pdfFont, "", 7.5)
	pdf.SetTextColor(198, 220, 233)
	pdf.Cell(105, 5, "Уточнит доступность, порядок оформления и условия сделки")
	pdf.SetXY(145, y+7)
	pdf.SetFont(pdfFont, "B", 9)
	pdf.SetTextColor(245, 166, 35)
	pdf.CellFormat(42, 6, "ДСК", "", 0, "R", false, 0, "")
	pdf.SetY(y + 28)
}

func sectionTitle(pdf *fpdf.Fpdf, title string) {
	if pdf.GetY() > 258 {
		pdf.AddPage()
	}
	pdf.SetFont(pdfFont, "B", 13)
	pdf.SetTextColor(12, 31, 43)
	pdf.Cell(0, 7, title)
	pdf.Ln(10)
}

func drawDSKMark(pdf *fpdf.Fpdf, x, y float64) {
	pdf.SetDrawColor(207, 215, 218)
	pdf.SetFillColor(245, 157, 35)
	pdf.Polygon([]fpdf.PointType{{X: x, Y: y + 5}, {X: x + 8, Y: y}, {X: x + 16, Y: y + 5}, {X: x + 8, Y: y + 10}}, "DF")
	pdf.SetFillColor(231, 235, 236)
	pdf.Polygon([]fpdf.PointType{{X: x, Y: y + 5}, {X: x + 8, Y: y + 10}, {X: x + 8, Y: y + 19}, {X: x, Y: y + 14}}, "DF")
	pdf.SetFillColor(37, 166, 223)
	pdf.Polygon([]fpdf.PointType{{X: x + 8, Y: y + 10}, {X: x + 16, Y: y + 5}, {X: x + 16, Y: y + 14}, {X: x + 8, Y: y + 19}}, "DF")
	pdf.SetDrawColor(255, 255, 255)
	pdf.SetLineWidth(0.7)
	pdf.Line(x+2.5, y+10, x+5.5, y+11.8)
	pdf.Line(x+10.5, y+13.1, x+14, y+11)
	pdf.SetXY(x+20, y+2)
	pdf.SetFont(pdfFont, "B", 18)
	pdf.SetTextColor(8, 55, 91)
	pdf.Cell(25, 10, "ДСК")
}

func setPDFColor(pdf *fpdf.Fpdf, name string) {
	if name == "ink" {
		pdf.SetTextColor(12, 31, 43)
	}
}

func formatMoney(value int64) string {
	negative := value < 0
	if negative {
		value = -value
	}
	digits := strconv.FormatInt(value, 10)
	for index := len(digits) - 3; index > 0; index -= 3 {
		digits = digits[:index] + " " + digits[index:]
	}
	if negative {
		digits = "− " + digits
	}
	return digits + " руб."
}

func formatArea(value float64) string {
	return strings.ReplaceAll(strconv.FormatFloat(value, 'f', 1, 64), ".", ",")
}

func optionalArea(value *float64) string {
	if value == nil {
		return "Дополнительный объект"
	}
	return formatArea(*value) + " м²"
}

func optionalUnitNumber(number *string, id *int) string {
	if number != nil && strings.TrimSpace(*number) != "" {
		return *number
	}
	if id == nil {
		return ""
	}
	return "№" + strconv.Itoa(*id)
}

func finishingLabel(value domain.FinishingType) string {
	switch value {
	case domain.FinishingTypeRough:
		return "Без отделки"
	case domain.FinishingTypeWhiteBox:
		return "White box"
	case domain.FinishingTypeTurnkey:
		return "Под ключ"
	default:
		return "Уточняется"
	}
}

func readinessLabel(value *int) string {
	if value == nil {
		return "Уточняется"
	}
	return strconv.Itoa(*value) + "%"
}

func russianDate(value time.Time) string {
	months := [...]string{"января", "февраля", "марта", "апреля", "мая", "июня", "июля", "августа", "сентября", "октября", "ноября", "декабря"}
	return fmt.Sprintf("%d %s %d", value.Day(), months[value.Month()-1], value.Year())
}

func normalizeOfferText(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	value = strings.Map(func(char rune) rune {
		if char == '\t' {
			return ' '
		}
		if unicode.IsControl(char) && char != '\n' {
			return -1
		}
		return char
	}, value)
	lines := strings.Split(value, "\n")
	for index, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimLeft(line, "#*")
		lines[index] = strings.TrimSpace(line)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func nonEmpty(values ...string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			result = append(result, value)
		}
	}
	return result
}

func valueOrZero(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}
