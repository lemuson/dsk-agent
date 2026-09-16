
BEGIN;

TRUNCATE TABLE messages, chat_session_rejections, chat_sessions, offers, offer_documents, offer_deliveries, deals, notifications, ai_audit_log, discount_policies CASCADE;

INSERT INTO users (id, name, email, password_hash, role, budget_max, preferences) VALUES
    (1001, 'Елена Соколова', 'supervisor@dsk.demo', '$2a$10$yvWiRVYYE6BbkVZmz3EFM.7g.90jRc6c9TWMP5pnRUVGuQNo42fNe', 'supervisor', NULL, '{}'::jsonb),
    (1002, 'Михаил Петров', 'manager@dsk.demo', '$2a$10$yvWiRVYYE6BbkVZmz3EFM.7g.90jRc6c9TWMP5pnRUVGuQNo42fNe', 'manager', NULL, '{}'::jsonb)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name, email = EXCLUDED.email, password_hash = EXCLUDED.password_hash,
    role = EXCLUDED.role, budget_max = EXCLUDED.budget_max, preferences = EXCLUDED.preferences;

DELETE FROM users WHERE role NOT IN ('manager', 'supervisor');

INSERT INTO residential_complexes (id, name, address, description) VALUES
    (2001, 'Квартал «Северный»', 'Воронеж, Московский проспект, 126', 'Семейный квартал с закрытыми дворами и подземным паркингом.'),
    (2002, 'ЖК «Левобережный»', 'Воронеж, Ленинский проспект, 215', 'Проект у водохранилища с готовой социальной инфраструктурой.'),
    (2003, 'ЖК «Горизонт»', 'Воронеж, ул. Шишкова, 142', 'Новый современный квартал комфорт-класса от ДСК.')
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, address = EXCLUDED.address, description = EXCLUDED.description;

INSERT INTO buildings (
    id, residential_complex_id, address, district, latitude, longitude, floors_count,
    planned_date, actual_date, status, type_wall_material,
    readiness_percent, forecast_date, delivery_shift_days
) VALUES

    (2101, 2001, 'д. 126/1', 'Коминтерновский', 51.70510000, 39.16820000, 17, CURRENT_DATE + 210, NULL, 'construction', 'panel', 68, CURRENT_DATE + 219, 0),
    (2102, 2001, 'д. 126/2', 'Коминтерновский', 51.70600000, 39.16910000, 20, CURRENT_DATE + 420, NULL, 'construction', 'monolith', 34, CURRENT_DATE + 420, 0),
    (2103, 2001, 'д. 126/3', 'Коминтерновский', 51.70700000, 39.17000000, 19, CURRENT_DATE + 550, NULL, 'construction', 'panel', 20, CURRENT_DATE + 550, 0),

    (2104, 2002, 'д. 215/1', 'Левобережный', 51.66090000, 39.28010000, 16, CURRENT_DATE - 90, CURRENT_DATE - 82, 'completed', 'brick', 100, CURRENT_DATE - 82, 0),
    (2105, 2002, 'д. 215/2', 'Левобережный', 51.66170000, 39.28100000, 18, CURRENT_DATE + 330, NULL, 'construction', 'panel', 45, CURRENT_DATE + 330, 0),
    (2106, 2002, 'д. 215/3', 'Левобережный', 51.66250000, 39.28200000, 16, CURRENT_DATE + 480, NULL, 'construction', 'brick', 15, CURRENT_DATE + 480, 0),

    (2107, 2003, 'д. 142/1', 'Центральный', 51.69250000, 39.20510000, 24, CURRENT_DATE + 600, NULL, 'construction', 'monolith', 28, CURRENT_DATE + 600, 0),
    (2108, 2003, 'д. 142/2', 'Центральный', 51.69350000, 39.20600000, 24, CURRENT_DATE + 720, NULL, 'construction', 'monolith', 10, CURRENT_DATE + 720, 0),
    (2109, 2003, 'д. 142/3', 'Центральный', 51.69450000, 39.20700000, 22, CURRENT_DATE + 850, NULL, 'design', 'monolith', 5, CURRENT_DATE + 850, 0)
ON CONFLICT (id) DO UPDATE SET
    residential_complex_id = EXCLUDED.residential_complex_id, address = EXCLUDED.address,
    district = EXCLUDED.district, latitude = EXCLUDED.latitude, longitude = EXCLUDED.longitude,
    floors_count = EXCLUDED.floors_count, planned_date = EXCLUDED.planned_date,
    actual_date = EXCLUDED.actual_date, status = EXCLUDED.status,
    type_wall_material = EXCLUDED.type_wall_material, readiness_percent = EXCLUDED.readiness_percent,
    forecast_date = EXCLUDED.forecast_date, delivery_shift_days = EXCLUDED.delivery_shift_days;

INSERT INTO apartments (id, building_id, number, rooms, floor, area, price, type_finishing, status) VALUES

    (3001, 2101, '181', 2, 10, 62.10, 8950000, 'white_box', 'free'),
    (3002, 2101, '182', 1, 10, 41.60, 6430000, 'turnkey', 'free'),
    (3003, 2101, '183', 3, 10, 78.40, 11250000, 'rough', 'free'),
    (3004, 2101, '184', 2, 10, 59.20, 8470000, 'white_box', 'free'),
    (3005, 2101, '185', 1, 10, 38.70, 5980000, 'rough', 'free'),
    (3006, 2101, '186', 2, 10, 64.80, 9180000, 'turnkey', 'free'),
    (3011, 2101, '127', 1, 7, 39.10, 6120000, 'rough', 'free'),
    (3012, 2101, '128', 3, 7, 81.30, 11860000, 'turnkey', 'free'),

    (3021, 2102, '74', 2, 5, 57.90, 8040000, 'white_box', 'free'),
    (3022, 2102, '75', 3, 5, 84.20, 12190000, 'rough', 'free'),
    (3023, 2102, '76', 1, 6, 40.50, 6250000, 'turnkey', 'free'),
    (3024, 2102, '77', 2, 6, 61.30, 8680000, 'white_box', 'free'),

    (3025, 2103, '12', 1, 2, 38.20, 5820000, 'rough', 'free'),
    (3026, 2103, '13', 2, 2, 56.40, 7920000, 'white_box', 'free'),
    (3027, 2103, '14', 3, 3, 79.50, 11150000, 'turnkey', 'free'),

    (3031, 2104, '46', 1, 3, 42.30, 6770000, 'turnkey', 'free'),
    (3032, 2104, '47', 2, 3, 63.80, 8920000, 'turnkey', 'free'),
    (3033, 2104, '48', 3, 4, 82.00, 11750000, 'turnkey', 'free'),

    (3051, 2105, '91', 2, 6, 60.40, 7980000, 'rough', 'free'),
    (3052, 2105, '92', 1, 6, 39.80, 5990000, 'white_box', 'free'),
    (3053, 2105, '93', 3, 7, 85.10, 11950000, 'turnkey', 'free'),

    (3054, 2106, '24', 1, 4, 41.20, 6180000, 'rough', 'free'),
    (3055, 2106, '25', 2, 4, 58.70, 8150000, 'white_box', 'free'),

    (3041, 2107, '201', 2, 12, 61.70, 9230000, 'white_box', 'free'),
    (3042, 2107, '202', 1, 12, 43.10, 6850000, 'turnkey', 'free'),
    (3043, 2107, '203', 3, 13, 88.60, 13450000, 'turnkey', 'free'),

    (3044, 2108, '55', 1, 5, 39.50, 6320000, 'rough', 'free'),
    (3045, 2108, '56', 2, 5, 62.80, 9380000, 'white_box', 'free'),

    (3046, 2109, '108', 2, 10, 65.20, 9650000, 'white_box', 'free'),
    (3047, 2109, '109', 3, 10, 91.40, 13900000, 'turnkey', 'free')
ON CONFLICT (id) DO UPDATE SET
    building_id = EXCLUDED.building_id, number = EXCLUDED.number, rooms = EXCLUDED.rooms,
    floor = EXCLUDED.floor, area = EXCLUDED.area, price = EXCLUDED.price,
    type_finishing = EXCLUDED.type_finishing, status = EXCLUDED.status;

INSERT INTO construction_progress (
    id, building_id, stage_name, planned_start_date, actual_start_date,
    planned_end_date, actual_end_date, status, completion_percentage,
    delay_reason, risk_level, delay_days
) VALUES
    (4001, 2101, 'excavation', CURRENT_DATE - 420, CURRENT_DATE - 420, CURRENT_DATE - 380, CURRENT_DATE - 378, 'completed', 100, NULL, 'low', 0),
    (4002, 2101, 'foundation', CURRENT_DATE - 378, CURRENT_DATE - 376, CURRENT_DATE - 300, CURRENT_DATE - 294, 'completed', 100, NULL, 'low', 0),
    (4003, 2101, 'frame', CURRENT_DATE - 292, CURRENT_DATE - 292, CURRENT_DATE - 80, CURRENT_DATE - 72, 'completed', 100, NULL, 'low', 0),
    (4004, 2101, 'roofing', CURRENT_DATE - 70, CURRENT_DATE - 68, CURRENT_DATE + 15, NULL, 'in_progress', 74, NULL, 'low', 0),
    (4005, 2101, 'finishing', CURRENT_DATE + 5, NULL, CURRENT_DATE + 170, NULL, 'not_started', 0, NULL, 'low', 0),
    (4011, 2102, 'excavation', CURRENT_DATE - 160, CURRENT_DATE - 160, CURRENT_DATE - 110, CURRENT_DATE - 108, 'completed', 100, NULL, 'low', 0),
    (4012, 2102, 'foundation', CURRENT_DATE - 108, CURRENT_DATE - 106, CURRENT_DATE - 30, NULL, 'in_progress', 82, NULL, 'low', 0),
    (4021, 2104, 'finishing', CURRENT_DATE - 240, CURRENT_DATE - 238, CURRENT_DATE - 110, CURRENT_DATE - 90, 'completed', 100, NULL, 'low', 0),
    (4031, 2107, 'excavation', CURRENT_DATE + 210, NULL, CURRENT_DATE + 270, NULL, 'not_started', 0, NULL, 'low', 0),
    (4041, 2105, 'frame', CURRENT_DATE - 190, CURRENT_DATE - 188, CURRENT_DATE - 20, NULL, 'in_progress', 46, NULL, 'low', 0)
ON CONFLICT (id) DO UPDATE SET
    building_id = EXCLUDED.building_id, stage_name = EXCLUDED.stage_name,
    planned_start_date = EXCLUDED.planned_start_date, actual_start_date = EXCLUDED.actual_start_date,
    planned_end_date = EXCLUDED.planned_end_date, actual_end_date = EXCLUDED.actual_end_date,
    status = EXCLUDED.status, completion_percentage = EXCLUDED.completion_percentage,
    delay_reason = EXCLUDED.delay_reason, risk_level = EXCLUDED.risk_level, delay_days = EXCLUDED.delay_days;

INSERT INTO ancillary_units (id, building_id, kind, number, area, price, status) VALUES
    (8001, 2101, 'parking', 'P-041', 13.50, 1250000, 'free'),
    (8002, 2101, 'parking', 'P-042', 13.50, 1250000, 'free'),
    (8003, 2101, 'storage', 'K-018', 4.80, 480000, 'free'),
    (8004, 2101, 'storage', 'K-019', 5.20, 530000, 'free'),
    (8011, 2102, 'parking', 'P-101', 14.10, 1390000, 'free'),
    (8012, 2102, 'storage', 'K-051', 6.00, 610000, 'free'),
    (8021, 2107, 'parking', 'P-201', 14.50, 1450000, 'free'),
    (8022, 2107, 'storage', 'K-201', 5.00, 520000, 'free')
ON CONFLICT (id) DO UPDATE SET
    building_id = EXCLUDED.building_id, kind = EXCLUDED.kind, number = EXCLUDED.number,
    area = EXCLUDED.area, price = EXCLUDED.price, status = EXCLUDED.status, updated_at = CURRENT_TIMESTAMP;

INSERT INTO discount_policies (
    id, building_id, role, max_discount_percent, version,
    valid_from, valid_to, created_by, created_at
) VALUES
    (8101, 2101, 'manager', 5.00, 1, CURRENT_TIMESTAMP - INTERVAL '30 days', NULL, 1001, CURRENT_TIMESTAMP - INTERVAL '30 days'),
    (8102, 2101, 'supervisor', 12.00, 1, CURRENT_TIMESTAMP - INTERVAL '30 days', NULL, 1001, CURRENT_TIMESTAMP - INTERVAL '30 days'),
    (8111, 2102, 'manager', 5.00, 1, CURRENT_TIMESTAMP - INTERVAL '30 days', NULL, 1001, CURRENT_TIMESTAMP - INTERVAL '30 days'),
    (8112, 2102, 'supervisor', 12.00, 1, CURRENT_TIMESTAMP - INTERVAL '30 days', NULL, 1001, CURRENT_TIMESTAMP - INTERVAL '30 days'),
    (8113, 2107, 'manager', 5.00, 1, CURRENT_TIMESTAMP - INTERVAL '30 days', NULL, 1001, CURRENT_TIMESTAMP - INTERVAL '30 days'),
    (8114, 2107, 'supervisor', 12.00, 1, CURRENT_TIMESTAMP - INTERVAL '30 days', NULL, 1001, CURRENT_TIMESTAMP - INTERVAL '30 days')
ON CONFLICT (id) DO UPDATE SET
    building_id = EXCLUDED.building_id, role = EXCLUDED.role,
    max_discount_percent = EXCLUDED.max_discount_percent, version = EXCLUDED.version,
    valid_from = EXCLUDED.valid_from, valid_to = EXCLUDED.valid_to,
    created_by = EXCLUDED.created_by, created_at = EXCLUDED.created_at;

INSERT INTO competitors (
    id, project_name, district, price_per_sqm, advantages, disadvantages,
    source_url, observed_at, updated_at, rooms, area
) VALUES

    (9001, 'ЖК «Современник»', 'Коминтерновский', 148000,
     'Рядом ТРЦ Московский проспект, развитая коммерческая инфраструктура, детские площадки премиум-класса',
     'Высокая плотность застройки, острый дефицит парковочных мест во дворе, сдача только в 4 кв. 2027 г.',
     'https://наш.дом.рф/каталог-новостроек/объект/48201',
     CURRENT_TIMESTAMP - INTERVAL '2 days', CURRENT_TIMESTAMP - INTERVAL '2 days', 1, 38.50),
    (9002, 'ЖК «Ботанический сад»', 'Коминтерновский', 139000,
     'Экологически чистый микрорайон, благоустроенный парк в шаговой доступности, чистый воздух',
     'Узкий выезд на Московский проспект, систематические утренние пробки, панельные межквартирные перекрытия с низкой звукоизоляцией',
     'https://наш.дом.рф/каталог-новостроек/объект/49112',
     CURRENT_TIMESTAMP - INTERVAL '1 day', CURRENT_TIMESTAMP - INTERVAL '1 day', 2, 59.00),
    (9003, 'ЖК «Грин Парк»', 'Коминтерновский', 141000,
     'Хвойный лесопарковый массив, чистый воздух, тихая закрытая территория',
     'Отсутствие школы в шаговой доступности, удаленность от центра города (более 40 минут на общественном транспорте)',
     'https://domclick.ru/complex/green-park-vrn',
     CURRENT_TIMESTAMP - INTERVAL '3 days', CURRENT_TIMESTAMP - INTERVAL '3 days', 3, 78.00),
    (9004, 'ЖК «Аксиома Север»', 'Коминтерновский', 136000,
     'Цена за квадратный метр на 4-6% ниже среднерыночной по району, гибкие условия рассрочки',
     'Черновая отделка без электроразводки и стяжки пола (потребует от 1.5 млн на ремонт), задержки по предыдущим очередям до 6 месяцев',
     'https://domclick.ru/complex/vrn-aksioma-sever',
     CURRENT_TIMESTAMP - INTERVAL '1 day', CURRENT_TIMESTAMP - INTERVAL '1 day', 2, 63.50),
    (9005, 'ЖК «Бульвар Победы»', 'Коминтерновский', 151000,
     'Просторные планировки евро-формата, большие кухни-гостиные от 18 кв.м',
     'Высокий тариф управляющей компании, отсутствие подземного паркинга, высокий уровень шума от магистрали',
     'https://наш.дом.рф/каталог-новостроек/объект/50341',
     CURRENT_TIMESTAMP - INTERVAL '4 days', CURRENT_TIMESTAMP - INTERVAL '4 days', 1, 41.20),

    (9011, 'ЖК «Дельфин»', 'Левобережный', 134000,
     'Живописный вид на Воронежское водохранилище, обновленный парк «Дельфин» в 5 минутах пешком',
     'Сложности с поиском парковочного места в вечернее время, панельное домостроение, слышимость соседей',
     'https://наш.дом.рф/каталог-новостроек/объект/39812',
     CURRENT_TIMESTAMP - INTERVAL '2 days', CURRENT_TIMESTAMP - INTERVAL '2 days', 2, 60.00),
    (9012, 'ЖК «Озерки»', 'Левобережный', 146000,
     'Доступный бюджет покупки, построена новая современная школа и два детских сада внутри района',
     'Значительная удаленность от исторического центра, близость промзоны, перегруженная транспортная развязка на выезде',
     'https://domclick.ru/complex/ozerki-vrn',
     CURRENT_TIMESTAMP - INTERVAL '1 day', CURRENT_TIMESTAMP - INTERVAL '1 day', 1, 37.00),
    (9013, 'ЖК «Мандарин»', 'Левобережный', 131000,
     'Яркий архитектурный облик, компактные и экономичные по бюджету планировки для молодых семей',
     'Небольшая площадь кухонь (до 8.5 м²), отсутствие индивидуальных кладовых, крышная котельная с повышенным тарифом',
     'https://domclick.ru/complex/vrn-mandarin',
     CURRENT_TIMESTAMP - INTERVAL '5 days', CURRENT_TIMESTAMP - INTERVAL '5 days', 2, 54.50),
    (9014, 'ЖК «Лазурный»', 'Левобережный', 138000,
     'Благоустроенная собственная набережная, прогулочный променад вдоль береговой линии',
     'Повышенная ветровая нагрузка в осенне-зимний период, штукатурный фасад требует регулярного обслуживания, класс энергоэффективности B',
     'https://наш.дом.рф/каталог-новостроек/объект/41209',
     CURRENT_TIMESTAMP - INTERVAL '3 days', CURRENT_TIMESTAMP - INTERVAL '3 days', 3, 74.00),
    (9015, 'ЖК «Левый берег»', 'Левобережный', 149000,
     'Удобный прямой выезд на Ленинский проспект, остановка транспорта в 2 минутах от дома',
     'Точечная застройка с минимальной придомовой территорией, нет закрытого двора без машин, окна выходят на дорогу',
     'https://domclick.ru/complex/left-bank-vrn',
     CURRENT_TIMESTAMP - INTERVAL '2 days', CURRENT_TIMESTAMP - INTERVAL '2 days', 1, 39.00),

    (9021, 'ЖК «Атлант»', 'Центральный', 157000,
     'Престижная центральная локация, панорамное остекление балконов и лоджий с видом на город',
     'Высокая этажность (24 этажа) при 2 лифтах на секцию (длительное ожидание), машиноместо в паркинге от 2.5 млн руб.',
     'https://наш.дом.рф/каталог-новостроек/объект/51902',
     CURRENT_TIMESTAMP - INTERVAL '1 day', CURRENT_TIMESTAMP - INTERVAL '1 day', 1, 44.00),
    (9022, 'ЖК «Европейский»', 'Центральный', 146000,
     'Европейский дизайн фасадов, консьерж-сервис, дизайнерские лобби с колясочными и лапомойками',
     'Сроки сдачи переносились на 9 месяцев, коммунальные платежи за содержание дома и территории на 30% выше средних',
     'https://domclick.ru/complex/european-vrn',
     CURRENT_TIMESTAMP - INTERVAL '2 days', CURRENT_TIMESTAMP - INTERVAL '2 days', 2, 68.00),
    (9023, 'ЖК «Сердце Города»', 'Центральный', 153000,
     'Закрытая охраняемая территория, дизайнерские входные группы, премиальные детские городки',
     'Квартиры сдаются только в черновой отделке (без стяжки и штукатурки), ключи только через 2 года',
     'https://наш.дом.рф/каталог-новостроек/объект/52814',
     CURRENT_TIMESTAMP - INTERVAL '3 days', CURRENT_TIMESTAMP - INTERVAL '3 days', 3, 88.00),
    (9024, 'ЖК «Финист»', 'Центральный', 147500,
     'Пешая доступность до драматического театра, университетов, парка «Орленок» и площади Ленина',
     'Высокая загруженность и загазованность прилегающих улиц (Кольцовская/Плехановская), ограниченная площадь детской площадки',
     'https://domclick.ru/complex/finist-vrn',
     CURRENT_TIMESTAMP - INTERVAL '2 days', CURRENT_TIMESTAMP - INTERVAL '2 days', 2, 65.00),
    (9025, 'ЖК «Петровский квартал»', 'Центральный', 156000,
     'Система «Умный дом», видеонаблюдение 360° по всему периметру, бесключевой доступ',
     'Высота потолков 2.65 м (в ЖК Горизонт от ДСК — 2.80 м), постоянные заторы на выезде в сторону проспекта Революции в часы пик',
     'https://наш.дом.рф/каталог-новостроек/объект/53190',
     CURRENT_TIMESTAMP - INTERVAL '1 day', CURRENT_TIMESTAMP - INTERVAL '1 day', 1, 42.50)
ON CONFLICT (id) DO UPDATE SET
    project_name = EXCLUDED.project_name, district = EXCLUDED.district,
    price_per_sqm = EXCLUDED.price_per_sqm, advantages = EXCLUDED.advantages,
    disadvantages = EXCLUDED.disadvantages, source_url = EXCLUDED.source_url,
    observed_at = EXCLUDED.observed_at, updated_at = EXCLUDED.updated_at,
    rooms = EXCLUDED.rooms, area = EXCLUDED.area;

SELECT setval(pg_get_serial_sequence('users', 'id'), COALESCE((SELECT MAX(id) FROM users), 1));
SELECT setval(pg_get_serial_sequence('residential_complexes', 'id'), COALESCE((SELECT MAX(id) FROM residential_complexes), 1));
SELECT setval(pg_get_serial_sequence('buildings', 'id'), COALESCE((SELECT MAX(id) FROM buildings), 1));
SELECT setval(pg_get_serial_sequence('apartments', 'id'), COALESCE((SELECT MAX(id) FROM apartments), 1));
SELECT setval(pg_get_serial_sequence('construction_progress', 'id'), COALESCE((SELECT MAX(id) FROM construction_progress), 1));
SELECT setval(pg_get_serial_sequence('ancillary_units', 'id'), COALESCE((SELECT MAX(id) FROM ancillary_units), 1));
SELECT setval(pg_get_serial_sequence('discount_policies', 'id'), COALESCE((SELECT MAX(id) FROM discount_policies), 1));
SELECT setval(pg_get_serial_sequence('competitors', 'id'), COALESCE((SELECT MAX(id) FROM competitors), 1));

COMMIT;
