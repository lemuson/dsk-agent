package swagger

import "encoding/json"

type openAPIDocument map[string]any
type openAPIObject map[string]any

func init() {
	UserSpec = updateUserSpec(UserSpec)
	StaffSpec = updateStaffSpec(StaffSpec)
}

func updateUserSpec(raw []byte) []byte {
	doc := decodeSpec(raw)
	installSharedSchemas(doc)
	paths := object(doc, "paths")

	delete(paths, "/api/ai/chat")

	paths["/api/notifications"] = openAPIObject{
		"get": securedListOperation("Notifications", "Получить уведомления пользователя", "Notification"),
	}
	operation(paths, "/api/notifications", "get")["parameters"] = []any{
		openAPIObject{"name": "unread", "in": "query", "required": false, "schema": openAPIObject{"type": "boolean"}},
	}
	paths["/api/notifications/read-all"] = openAPIObject{
		"put": openAPIObject{
			"tags": []string{"Notifications"}, "summary": "Отметить все уведомления как прочитанные", "security": bearerSecurity(),
			"responses": responses(map[string]string{"200": ""}),
		},
	}
	paths["/api/notifications/{id}/read"] = openAPIObject{
		"put": openAPIObject{
			"tags": []string{"Notifications"}, "summary": "Отметить уведомление как прочитанное", "security": bearerSecurity(),
			"parameters": []any{idParameter("id")},
			"responses":  responses(map[string]string{"200": "", "404": ""}),
		},
	}

	paths["/api/progress/{id}"] = openAPIObject{
		"get": securedReadOperation("Catalog", "Получить этап строительства", "ConstructionProgress"),
	}

	delete(operation(paths, "/api/progress/{id}", "get"), "security")
	installOfferReadRoutes(paths)
	paths["/api/buildings/{buildingId}/ancillary-units"] = openAPIObject{
		"get": securedListOperationWithParameter("Catalog", "Получить парковки и кладовые корпуса", "AncillaryUnit", "buildingId"),
	}
	delete(operation(paths, "/api/buildings/{buildingId}/ancillary-units", "get"), "security")
	return encodeSpec(doc)
}

func updateStaffSpec(raw []byte) []byte {
	doc := decodeSpec(raw)
	installSharedSchemas(doc)
	paths := object(doc, "paths")
	staffRegister := operation(paths, "/api/auth/register", "post")
	staffRegister["summary"] = "Создать сотрудника (только руководитель)"
	staffRegister["security"] = bearerSecurity()

	paths["/api/auth/logout"] = openAPIObject{
		"post": openAPIObject{"tags": []string{"Auth"}, "summary": "Выход из системы", "responses": responses(map[string]string{"200": ""})},
	}
	paths["/api/users/{id}"] = openAPIObject{
		"get": securedReadOperation("Users", "Получить пользователя по ID", "User"),
	}

	ai := operation(paths, "/api/ai/chat", "post")
	ai["security"] = bearerSecurity()
	ai["requestBody"] = requestBody("AIChatRequest")
	ai["responses"] = responses(map[string]string{
		"200": "AIChatResponse", "400": "", "401": "", "403": "", "500": "", "503": "", "504": "",
	})

	installStaffConstructionRoutes(paths)
	installStaffSalesRoutes(paths)
	return encodeSpec(doc)
}

func installOfferReadRoutes(paths openAPIObject) {
	paths["/api/offers"] = openAPIObject{
		"get": securedListOperation("Offers", "Получить доступные коммерческие предложения", "Offer"),
	}
	paths["/api/offers/{id}"] = openAPIObject{
		"get": securedReadOperation("Offers", "Получить коммерческое предложение", "Offer"),
	}
	paths["/api/offers/{id}/pdf"] = openAPIObject{
		"get": securedReadOperation("Offers", "Скачать утверждённое КП в PDF", ""),
	}
}

func installStaffSalesRoutes(paths openAPIObject) {
	installOfferReadRoutes(paths)
	paths["/api/offers/calculate"] = openAPIObject{
		"post": securedWriteOperation("Offers", "Рассчитать КП по финансовой модели", "OfferCalculateRequest", "OfferCalculation", "200"),
	}
	paths["/api/offers"].(openAPIObject)["post"] = securedWriteOperation("Offers", "Создать версию КП", "OfferCreateRequest", "Offer", "201")
	for _, action := range []string{"approval-request", "approve", "reject"} {
		requestSchema := ""
		if action == "reject" {
			requestSchema = "OfferRejectRequest"
		}
		op := securedWriteOperation("Offers", "Изменить состояние согласования КП", requestSchema, "Offer", "200")
		op["parameters"] = []any{idParameter("id")}
		paths["/api/offers/{id}/"+action] = openAPIObject{"post": op}
	}
	paths["/api/offers/{id}/send"] = openAPIObject{
		"post": securedWriteOperation("Offers", "Отправить утверждённое КП клиенту и сохранить результат доставки", "", "OfferDelivery", "200"),
	}
	paths["/api/ai/dialog/analyze"] = openAPIObject{
		"post": securedWriteOperation("AI", "Извлечь подтверждённые факты из переписки", "DealAIRequest", "DialogAnalysis", "200"),
	}
	paths["/api/ai/dialog/reply-assist"] = openAPIObject{
		"post": securedWriteOperation("AI", "Подготовить приватный черновик ответа менеджеру", "ReplyAssistRequest", "ReplyAssist", "200"),
	}
	paths["/api/competitors"] = openAPIObject{
		"get":  securedListOperation("Competitors", "Получить наблюдения по конкурентам", "Competitor"),
		"post": securedWriteOperation("Competitors", "Добавить наблюдение", "CompetitorWrite", "Competitor", "201"),
	}
	paths["/api/competitors/{id}"] = openAPIObject{
		"put": securedWriteOperation("Competitors", "Обновить наблюдение", "CompetitorWrite", "Competitor", "200"),
	}
	paths["/api/buildings/{buildingId}/discount-policies"] = openAPIObject{
		"get": securedListOperationWithParameter("Discounts", "Получить версии матрицы скидок корпуса", "DiscountPolicy", "buildingId"),
	}
	paths["/api/discount-policies"] = openAPIObject{
		"post": securedWriteOperation("Discounts", "Создать новую версию лимита скидки", "DiscountPolicyWrite", "DiscountPolicy", "201"),
	}
	paths["/api/buildings/{buildingId}/ancillary-units"] = openAPIObject{
		"get": securedListOperationWithParameter("Catalog", "Получить парковки и кладовые корпуса", "AncillaryUnit", "buildingId"),
	}
	paths["/api/ancillary-units"] = openAPIObject{
		"post": securedWriteOperation("Catalog", "Добавить парковку или кладовую", "AncillaryUnitWrite", "AncillaryUnit", "201"),
	}
	paths["/api/ancillary-units/{id}"] = openAPIObject{
		"put": securedWriteOperation("Catalog", "Обновить парковку или кладовую", "AncillaryUnitWrite", "AncillaryUnit", "200"),
	}
	erpRead := securedReadOperation("ERP", "Получить производственную сводку корпуса", "ERPSnapshot")
	erpRead["parameters"] = []any{idParameter("buildingId")}
	paths["/api/buildings/{buildingId}/erp"] = openAPIObject{"get": erpRead}
	paths["/api/erp/events"] = openAPIObject{"post": securedWriteOperation("ERP", "Добавить производственное событие", "ERPEventWrite", "ERPEvent", "201")}
	paths["/api/erp/material-stocks"] = openAPIObject{"post": securedWriteOperation("ERP", "Сохранить остаток материала", "MaterialStockWrite", "MaterialStock", "200")}
	paths["/api/erp/production-schedules"] = openAPIObject{"post": securedWriteOperation("ERP", "Добавить производственную задачу", "ProductionScheduleWrite", "ProductionSchedule", "201")}
	paths["/api/reminders"] = openAPIObject{
		"get":  securedListOperation("Reminders", "Получить напоминания сотрудника или отдела", "StaffReminder"),
		"post": securedWriteOperation("Reminders", "Создать напоминание", "StaffReminderWrite", "StaffReminder", "201"),
	}
	paths["/api/reminders/{id}/complete"] = openAPIObject{"put": securedWriteOperation("Reminders", "Отметить напоминание выполненным", "", "StaffReminder", "200")}
}

func installStaffConstructionRoutes(paths openAPIObject) {
	paths["/api/complexes"] = openAPIObject{
		"get":  securedListOperation("Construction", "Получить жилые комплексы", "ResidentialComplex"),
		"post": securedWriteOperation("Construction", "Создать жилой комплекс", "ResidentialComplexWrite", "ResidentialComplex", "201"),
	}
	paths["/api/complexes/{id}"] = crudItemOperations("жилой комплекс", "ResidentialComplexWrite", "ResidentialComplex")
	paths["/api/complexes/{complexId}/buildings"] = openAPIObject{
		"get": securedListOperationWithParameter("Construction", "Получить корпуса жилого комплекса", "Building", "complexId"),
	}

	paths["/api/buildings"] = openAPIObject{
		"post": securedWriteOperation("Construction", "Создать корпус", "BuildingWrite", "Building", "201"),
	}
	paths["/api/buildings/{id}"] = crudItemOperations("корпус", "BuildingWrite", "Building")
	paths["/api/buildings/{buildingId}/apartments"] = openAPIObject{
		"get": securedListOperationWithParameter("Construction", "Получить квартиры корпуса", "Apartment", "buildingId"),
	}
	paths["/api/buildings/{buildingId}/progress"] = openAPIObject{
		"get": securedListOperationWithParameter("Construction", "Получить этапы строительства корпуса", "ConstructionProgress", "buildingId"),
	}

	paths["/api/apartments"] = openAPIObject{
		"post": securedWriteOperation("Construction", "Создать квартиру", "ApartmentWrite", "Apartment", "201"),
	}
	paths["/api/apartments/{id}"] = crudItemOperations("квартиру", "ApartmentWrite", "Apartment")

	paths["/api/progress"] = openAPIObject{
		"post": securedWriteOperation("Construction", "Создать этап строительства", "ConstructionProgressWrite", "ConstructionProgress", "201"),
	}
	paths["/api/progress/{id}"] = crudItemOperations("этап строительства", "ConstructionProgressWrite", "ConstructionProgress")
}

func installSharedSchemas(doc openAPIDocument) {
	components := object(doc, "components")
	schemas := object(components, "schemas")
	for name, schema := range sharedSchemas() {
		schemas[name] = schema
	}
}

func sharedSchemas() openAPIObject {
	integer := func(nullable bool) openAPIObject {
		result := openAPIObject{"type": "integer"}
		if nullable {
			result["nullable"] = true
		}
		return result
	}
	date := func() openAPIObject { return openAPIObject{"type": "string", "format": "date-time", "nullable": true} }
	enum := func(values ...string) openAPIObject { return openAPIObject{"type": "string", "enum": values} }

	return openAPIObject{
		"AIChatRequest": schema([]string{"message", "session_id"}, openAPIObject{
			"message": openAPIObject{"type": "string"}, "session_id": integer(false), "deal_id": integer(true),
			"parking_unit_id": integer(true), "storage_unit_id": integer(true),
		}),
		"AIChatResponse": schema([]string{"message", "agent", "intent"}, openAPIObject{
			"message": openAPIObject{"type": "string"}, "agent": openAPIObject{"type": "string"}, "intent": openAPIObject{"type": "string"},
		}),
		"CreateDealRequest": schema([]string{"id_user", "id_apartment", "percent_discount"}, openAPIObject{
			"id_user": integer(false), "id_apartment": integer(false), "id_chat_session": integer(true),
			"percent_discount": openAPIObject{"type": "number"},
		}),
		"User": schema([]string{"id", "name", "email", "role"}, openAPIObject{
			"id": integer(false), "name": openAPIObject{"type": "string"}, "email": openAPIObject{"type": "string", "format": "email"},
			"role": enum("user", "manager", "supervisor"), "budget_max": openAPIObject{"type": "integer", "format": "int64", "nullable": true},
			"preferences": openAPIObject{"type": "object", "additionalProperties": true},
		}),
		"ResidentialComplex": schema([]string{"id", "name", "address", "description"}, openAPIObject{
			"id": integer(false), "name": openAPIObject{"type": "string"}, "address": openAPIObject{"type": "string"}, "description": openAPIObject{"type": "string"},
		}),
		"ResidentialComplexWrite": schema([]string{"name", "address"}, openAPIObject{
			"name": openAPIObject{"type": "string"}, "address": openAPIObject{"type": "string"}, "description": openAPIObject{"type": "string"},
		}),
		"Building": schema([]string{"id", "residential_complex_id", "address", "district", "latitude", "longitude", "floors_count", "status", "type_wall_material"}, openAPIObject{
			"id": integer(false), "residential_complex_id": integer(false), "address": openAPIObject{"type": "string"}, "district": openAPIObject{"type": "string"},
			"latitude": openAPIObject{"type": "number", "format": "double"}, "longitude": openAPIObject{"type": "number", "format": "double"}, "floors_count": integer(false),
			"planned_date": date(), "actual_date": date(), "status": enum("design", "construction", "completed", "suspended"),
			"type_wall_material": enum("panel", "monolith", "brick", "block"),
		}),
		"BuildingWrite": schema([]string{"residential_complex_id", "address", "status", "type_wall_material"}, openAPIObject{
			"residential_complex_id": integer(false), "address": openAPIObject{"type": "string"}, "district": openAPIObject{"type": "string"},
			"latitude": openAPIObject{"type": "number", "format": "double"}, "longitude": openAPIObject{"type": "number", "format": "double"}, "floors_count": integer(false),
			"planned_date": date(), "actual_date": date(), "status": enum("design", "construction", "completed", "suspended"),
			"type_wall_material": enum("panel", "monolith", "brick", "block"),
		}),
		"Apartment": schema([]string{"id", "building_id", "number", "rooms", "floor", "area", "price", "type_finishing", "status"}, openAPIObject{
			"id": integer(false), "building_id": integer(false), "number": openAPIObject{"type": "string"}, "rooms": integer(false), "floor": integer(false),
			"area": openAPIObject{"type": "number"}, "price": openAPIObject{"type": "number"}, "type_finishing": enum("rough", "white_box", "turnkey"),
			"status": enum("free", "booked", "sold"),
		}),
		"ApartmentWrite": schema([]string{"building_id", "number", "rooms", "area", "price", "status"}, openAPIObject{
			"building_id": integer(false), "number": openAPIObject{"type": "string"}, "rooms": integer(false), "floor": integer(false),
			"area": openAPIObject{"type": "number"}, "price": openAPIObject{"type": "number"}, "type_finishing": enum("rough", "white_box", "turnkey"),
			"status": enum("free", "booked", "sold"),
		}),
		"ConstructionProgress": schema([]string{"id", "building_id", "stage_name", "status", "delay_reason"}, openAPIObject{
			"id": integer(false), "building_id": integer(false), "stage_name": enum("excavation", "foundation", "frame", "roofing", "finishing"),
			"planned_start_date": date(), "actual_start_date": date(), "planned_end_date": date(), "actual_end_date": date(),
			"status": enum("not_started", "in_progress", "completed", "delayed"), "completion_percentage": integer(true),
			"delay_reason": openAPIObject{"type": "string"}, "risk_level": openAPIObject{"type": "string", "nullable": true}, "delay_days": integer(true),
		}),
		"ConstructionProgressWrite": schema([]string{"building_id", "stage_name", "status"}, openAPIObject{
			"building_id": integer(false), "stage_name": enum("excavation", "foundation", "frame", "roofing", "finishing"),
			"planned_start_date": date(), "actual_start_date": date(), "planned_end_date": date(), "actual_end_date": date(),
			"status": enum("not_started", "in_progress", "completed", "delayed"), "completion_percentage": integer(true),
			"delay_reason": openAPIObject{"type": "string"}, "risk_level": openAPIObject{"type": "string", "nullable": true}, "delay_days": integer(true),
		}),
		"Offer": schema([]string{"id", "deal_id", "version", "base_price", "discount_percent", "final_price", "status"}, openAPIObject{
			"id": integer(false), "deal_id": integer(false), "version": integer(false), "created_by": integer(false), "base_price": integer(false),
			"discount_percent": openAPIObject{"type": "string"}, "final_price": integer(false), "generated_text": openAPIObject{"type": "string"},
			"parking_unit_id": integer(true), "parking_number": openAPIObject{"type": "string", "nullable": true}, "parking_price": integer(false),
			"storage_unit_id": integer(true), "storage_number": openAPIObject{"type": "string", "nullable": true}, "storage_price": integer(false),
			"status": enum("draft", "pending_approval", "approved", "rejected"), "approval_required": openAPIObject{"type": "boolean"},
		}),
		"OfferCalculateRequest":   schema([]string{"deal_id", "discount_percent"}, openAPIObject{"deal_id": integer(false), "discount_percent": openAPIObject{"type": "string"}, "parking_unit_id": integer(true), "storage_unit_id": integer(true)}),
		"OfferCreateRequest":      schema([]string{"request_id", "deal_id", "discount_percent", "generated_text"}, openAPIObject{"request_id": openAPIObject{"type": "string"}, "deal_id": integer(false), "discount_percent": openAPIObject{"type": "string"}, "generated_text": openAPIObject{"type": "string"}, "parking_unit_id": integer(true), "storage_unit_id": integer(true)}),
		"OfferCalculation":        schema([]string{"deal_id", "base_price", "apartment_price", "final_price", "max_allowed_discount", "requires_approval"}, openAPIObject{"deal_id": integer(false), "base_price": integer(false), "apartment_price": integer(false), "parking_unit_id": integer(true), "parking_number": openAPIObject{"type": "string", "nullable": true}, "parking_price": integer(false), "storage_unit_id": integer(true), "storage_number": openAPIObject{"type": "string", "nullable": true}, "storage_price": integer(false), "final_price": integer(false), "max_allowed_discount": openAPIObject{"type": "string"}, "requires_approval": openAPIObject{"type": "boolean"}}),
		"OfferRejectRequest":      schema([]string{"reason"}, openAPIObject{"reason": openAPIObject{"type": "string"}}),
		"DealAIRequest":           schema([]string{"deal_id"}, openAPIObject{"deal_id": integer(false)}),
		"ReplyAssistRequest":      schema([]string{"deal_id"}, openAPIObject{"deal_id": integer(false), "selected_text": openAPIObject{"type": "string", "nullable": true}}),
		"DialogAnalysis":          schema([]string{"deal_id", "client_id", "analysis"}, openAPIObject{"deal_id": integer(false), "client_id": integer(false), "analysis": openAPIObject{"type": "object", "additionalProperties": true}}),
		"ReplyAssist":             schema([]string{"deal_id", "suggested_reply"}, openAPIObject{"deal_id": integer(false), "suggested_reply": openAPIObject{"type": "string"}}),
		"Competitor":              schema([]string{"id", "project_name", "district"}, openAPIObject{"id": integer(false), "project_name": openAPIObject{"type": "string"}, "district": openAPIObject{"type": "string"}, "price_per_sqm": integer(true), "source_url": openAPIObject{"type": "string", "nullable": true}, "observed_at": date()}),
		"CompetitorWrite":         schema([]string{"project_name", "district"}, openAPIObject{"project_name": openAPIObject{"type": "string"}, "district": openAPIObject{"type": "string"}, "price_per_sqm": integer(true), "source_url": openAPIObject{"type": "string", "nullable": true}, "observed_at": date()}),
		"DiscountPolicy":          schema([]string{"id", "building_id", "role", "max_discount_percent", "version", "valid_from"}, openAPIObject{"id": integer(false), "building_id": integer(false), "role": enum("manager", "supervisor"), "max_discount_percent": openAPIObject{"type": "string"}, "version": integer(false), "valid_from": date(), "valid_to": date()}),
		"DiscountPolicyWrite":     schema([]string{"building_id", "role", "max_discount_percent", "valid_from"}, openAPIObject{"building_id": integer(false), "role": enum("manager", "supervisor"), "max_discount_percent": openAPIObject{"type": "string"}, "valid_from": date()}),
		"AncillaryUnit":           schema([]string{"id", "building_id", "kind", "number", "price", "status"}, openAPIObject{"id": integer(false), "building_id": integer(false), "kind": enum("parking", "storage"), "number": openAPIObject{"type": "string"}, "area": openAPIObject{"type": "number", "nullable": true}, "price": integer(false), "status": enum("free", "booked", "sold")}),
		"AncillaryUnitWrite":      schema([]string{"building_id", "kind", "number", "price", "status"}, openAPIObject{"building_id": integer(false), "kind": enum("parking", "storage"), "number": openAPIObject{"type": "string"}, "area": openAPIObject{"type": "number", "nullable": true}, "price": integer(false), "status": enum("free", "booked", "sold")}),
		"OfferDelivery":           schema([]string{"id", "offer_id", "recipient", "channel", "status"}, openAPIObject{"id": integer(false), "offer_id": integer(false), "recipient": openAPIObject{"type": "string"}, "channel": enum("email"), "status": enum("pending", "sent", "failed"), "error_message": openAPIObject{"type": "string", "nullable": true}}),
		"ERPEvent":                schema([]string{"id", "building_id", "kind", "title", "severity"}, openAPIObject{"id": integer(false), "building_id": integer(false), "kind": enum("schedule", "supply", "material", "project_change"), "title": openAPIObject{"type": "string"}, "details": openAPIObject{"type": "string"}, "severity": enum("low", "medium", "high"), "affects_delivery": openAPIObject{"type": "boolean"}, "delay_days": integer(true)}),
		"ERPEventWrite":           schema([]string{"building_id", "kind", "title", "details", "severity"}, openAPIObject{"building_id": integer(false), "kind": enum("schedule", "supply", "material", "project_change"), "title": openAPIObject{"type": "string"}, "details": openAPIObject{"type": "string"}, "severity": enum("low", "medium", "high"), "affects_delivery": openAPIObject{"type": "boolean"}, "delay_days": integer(true)}),
		"MaterialStock":           schema([]string{"id", "building_id", "material_name", "quantity", "unit", "minimum_quantity"}, openAPIObject{"id": integer(false), "building_id": integer(false), "material_name": openAPIObject{"type": "string"}, "quantity": openAPIObject{"type": "number"}, "unit": openAPIObject{"type": "string"}, "minimum_quantity": openAPIObject{"type": "number"}}),
		"MaterialStockWrite":      schema([]string{"building_id", "material_name", "quantity", "unit", "minimum_quantity"}, openAPIObject{"building_id": integer(false), "material_name": openAPIObject{"type": "string"}, "quantity": openAPIObject{"type": "number"}, "unit": openAPIObject{"type": "string"}, "minimum_quantity": openAPIObject{"type": "number"}}),
		"ProductionSchedule":      schema([]string{"id", "building_id", "product_name", "planned_quantity", "produced_quantity", "planned_date", "status"}, openAPIObject{"id": integer(false), "building_id": integer(false), "product_name": openAPIObject{"type": "string"}, "planned_quantity": integer(false), "produced_quantity": integer(false), "planned_date": date(), "status": enum("planned", "in_progress", "completed", "delayed")}),
		"ProductionScheduleWrite": schema([]string{"building_id", "product_name", "planned_quantity", "produced_quantity", "planned_date", "status"}, openAPIObject{"building_id": integer(false), "product_name": openAPIObject{"type": "string"}, "planned_quantity": integer(false), "produced_quantity": integer(false), "planned_date": date(), "status": enum("planned", "in_progress", "completed", "delayed")}),
		"ERPSnapshot":             schema([]string{"events", "material_stocks", "production_schedules"}, openAPIObject{"events": openAPIObject{"type": "array", "items": ref("ERPEvent")}, "material_stocks": openAPIObject{"type": "array", "items": ref("MaterialStock")}, "production_schedules": openAPIObject{"type": "array", "items": ref("ProductionSchedule")}}),
		"StaffReminder":           schema([]string{"id", "assigned_to", "created_by", "title", "due_at"}, openAPIObject{"id": integer(false), "assigned_to": integer(false), "created_by": integer(false), "deal_id": integer(true), "title": openAPIObject{"type": "string"}, "due_at": date(), "completed_at": date()}),
		"StaffReminderWrite":      schema([]string{"assigned_to", "title", "due_at"}, openAPIObject{"assigned_to": integer(false), "deal_id": integer(true), "title": openAPIObject{"type": "string"}, "due_at": date()}),
		"Notification": schema([]string{"id", "user_id", "type", "title", "message", "is_read", "created_at"}, openAPIObject{
			"id": integer(false), "user_id": integer(false), "deal_id": integer(true),
			"type":  enum("construction_delay", "construction_risk", "deal_update", "general"),
			"title": openAPIObject{"type": "string"}, "message": openAPIObject{"type": "string"},
			"is_read": openAPIObject{"type": "boolean"}, "created_at": date(), "read_at": date(),
		}),
	}
}

func schema(required []string, properties openAPIObject) openAPIObject {
	result := openAPIObject{"type": "object", "properties": properties}
	if len(required) > 0 {
		result["required"] = required
	}
	return result
}

func crudItemOperations(noun, requestSchema, responseSchema string) openAPIObject {
	update := securedWriteOperation("Construction", "Обновить "+noun, requestSchema, responseSchema, "200")
	update["description"] = "Доступно только руководителю отдела продаж"
	update["parameters"] = []any{idParameter("id")}
	return openAPIObject{
		"get": securedReadOperation("Construction", "Получить "+noun, responseSchema),
		"put": update,
	}
}

func securedReadOperation(tag, summary, responseSchema string) openAPIObject {
	return openAPIObject{
		"tags": []string{tag}, "summary": summary, "security": bearerSecurity(), "parameters": []any{idParameter("id")},
		"responses": responses(map[string]string{"200": responseSchema, "400": "", "401": "", "403": "", "404": ""}),
	}
}

func securedListOperation(tag, summary, itemSchema string) openAPIObject {
	return openAPIObject{
		"tags": []string{tag}, "summary": summary, "security": bearerSecurity(),
		"responses": responses(map[string]string{"200": "[]" + itemSchema, "401": "", "403": "", "500": ""}),
	}
}

func securedListOperationWithParameter(tag, summary, itemSchema, parameter string) openAPIObject {
	op := securedListOperation(tag, summary, itemSchema)
	op["parameters"] = []any{idParameter(parameter)}
	return op
}

func securedWriteOperation(tag, summary, requestSchema, responseSchema, successCode string) openAPIObject {
	result := openAPIObject{
		"tags": []string{tag}, "summary": summary, "security": bearerSecurity(),
		"responses": responses(map[string]string{successCode: responseSchema, "400": "", "401": "", "403": "", "500": ""}),
	}
	if requestSchema != "" {
		result["requestBody"] = requestBody(requestSchema)
	}
	return result
}

func securedDeleteOperation(tag, summary string) openAPIObject {
	return openAPIObject{
		"tags": []string{tag}, "summary": summary, "security": bearerSecurity(), "parameters": []any{idParameter("id")},
		"responses": responses(map[string]string{"204": "", "400": "", "401": "", "403": "", "500": ""}),
	}
}

func requestBody(schemaName string) openAPIObject {
	return openAPIObject{"required": true, "content": openAPIObject{"application/json": openAPIObject{"schema": ref(schemaName)}}}
}

func responses(items map[string]string) openAPIObject {
	result := openAPIObject{}
	for code, schemaName := range items {
		response := openAPIObject{"description": "HTTP " + code}
		if schemaName != "" {
			var responseSchema openAPIObject
			if len(schemaName) > 2 && schemaName[:2] == "[]" {
				responseSchema = openAPIObject{"type": "array", "items": ref(schemaName[2:])}
			} else {
				responseSchema = ref(schemaName)
			}
			response["content"] = openAPIObject{"application/json": openAPIObject{"schema": responseSchema}}
		}
		result[code] = response
	}
	return result
}

func idParameter(name string) openAPIObject {
	return openAPIObject{"name": name, "in": "path", "required": true, "schema": openAPIObject{"type": "integer"}}
}

func bearerSecurity() []any         { return []any{openAPIObject{"BearerAuth": []any{}}} }
func ref(name string) openAPIObject { return openAPIObject{"$ref": "#/components/schemas/" + name} }

func operation(paths openAPIObject, path, method string) openAPIObject {
	pathObject, ok := paths[path].(map[string]any)
	if !ok {
		pathObject = openAPIObject{}
		paths[path] = pathObject
	}
	op, ok := pathObject[method].(map[string]any)
	if !ok {
		op = openAPIObject{}
		pathObject[method] = op
	}
	return op
}

func object(parent map[string]any, key string) openAPIObject {
	if value, ok := parent[key].(map[string]any); ok {
		return value
	}
	value := openAPIObject{}
	parent[key] = value
	return value
}

func decodeSpec(raw []byte) openAPIDocument {
	var doc openAPIDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		panic("invalid embedded OpenAPI document: " + err.Error())
	}
	return doc
}

func encodeSpec(doc openAPIDocument) []byte {
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		panic("cannot encode OpenAPI document: " + err.Error())
	}
	return raw
}
