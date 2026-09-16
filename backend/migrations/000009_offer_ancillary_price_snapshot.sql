
ALTER TABLE offers
    ADD COLUMN IF NOT EXISTS parking_price BIGINT NOT NULL DEFAULT 0
        CHECK (parking_price >= 0),
    ADD COLUMN IF NOT EXISTS storage_price BIGINT NOT NULL DEFAULT 0
        CHECK (storage_price >= 0);

UPDATE offers AS offer
SET parking_price = unit.price
FROM ancillary_units AS unit
WHERE offer.parking_unit_id = unit.id
  AND offer.parking_price = 0;

UPDATE offers AS offer
SET storage_price = unit.price
FROM ancillary_units AS unit
WHERE offer.storage_unit_id = unit.id
  AND offer.storage_price = 0;
