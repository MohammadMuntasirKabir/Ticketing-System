-- +migrate Up
-- Airline Ticketing System Schema
-- PostgreSQL 18

-- =============================================
-- Reference Data
-- =============================================

CREATE TABLE IF NOT EXISTS airports (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    iata_code   VARCHAR(3) NOT NULL UNIQUE,
    name        VARCHAR(255) NOT NULL,
    city        VARCHAR(100) NOT NULL,
    country     VARCHAR(100) NOT NULL,
    timezone    VARCHAR(50) NOT NULL DEFAULT 'UTC',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS airlines (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    iata_code       VARCHAR(2) NOT NULL UNIQUE,
    name            VARCHAR(255) NOT NULL,
    logo_url        TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS aircraft (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    model           VARCHAR(100) NOT NULL,
    manufacturer    VARCHAR(100) NOT NULL,
    total_seats     INTEGER NOT NULL CHECK (total_seats > 0),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS cabin_classes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code            VARCHAR(20) NOT NULL UNIQUE,
    name            VARCHAR(100) NOT NULL,
    description     TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS fare_classes (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code                VARCHAR(20) NOT NULL UNIQUE,
    name                VARCHAR(100) NOT NULL,
    cabin_class_id      UUID NOT NULL REFERENCES cabin_classes(id),
    base_price          DECIMAL(10,2) NOT NULL DEFAULT 0,
    refundable          BOOLEAN NOT NULL DEFAULT FALSE,
    changeable          BOOLEAN NOT NULL DEFAULT FALSE,
    checked_bags        INTEGER NOT NULL DEFAULT 0,
    carry_on_bags       INTEGER NOT NULL DEFAULT 1,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- =============================================
-- Route & Schedule
-- =============================================

CREATE TABLE IF NOT EXISTS routes (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    airline_id          UUID NOT NULL REFERENCES airlines(id),
    aircraft_id         UUID NOT NULL REFERENCES aircraft(id),
    origin_id           UUID NOT NULL REFERENCES airports(id),
    destination_id      UUID NOT NULL REFERENCES airports(id),
    flight_number       VARCHAR(10) NOT NULL,
    duration_minutes    INTEGER NOT NULL CHECK (duration_minutes > 0),
    distance_miles      INTEGER NOT NULL DEFAULT 0,
    active              BOOLEAN NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (airline_id, flight_number)
);

CREATE INDEX idx_routes_origin ON routes(origin_id);
CREATE INDEX idx_routes_destination ON routes(destination_id);
CREATE INDEX idx_routes_airline ON routes(airline_id);

CREATE TABLE IF NOT EXISTS flight_schedules (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    route_id            UUID NOT NULL REFERENCES routes(id),
    departure_time      TIME NOT NULL,
    arrival_time        TIME NOT NULL,
    days_of_week        INTEGER[] NOT NULL DEFAULT '{}',
    active              BOOLEAN NOT NULL DEFAULT TRUE,
    effective_from      DATE NOT NULL DEFAULT CURRENT_DATE,
    effective_until     DATE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_flight_schedules_route ON flight_schedules(route_id);

CREATE TABLE IF NOT EXISTS flight_instances (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    schedule_id         UUID NOT NULL REFERENCES flight_schedules(id),
    route_id            UUID NOT NULL REFERENCES routes(id),
    aircraft_id         UUID NOT NULL REFERENCES aircraft(id),
    departure_date      DATE NOT NULL,
    departure_time      TIMESTAMPTZ NOT NULL,
    arrival_time        TIMESTAMPTZ NOT NULL,
    status              VARCHAR(20) NOT NULL DEFAULT 'scheduled'
                        CHECK (status IN ('scheduled', 'boarding', 'departed', 'arrived', 'cancelled', 'delayed')),
    gate                VARCHAR(10),
    terminal            VARCHAR(10),
    available_seats     INTEGER NOT NULL DEFAULT 0,
    price_modifier      DECIMAL(4,2) NOT NULL DEFAULT 1.00,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (schedule_id, departure_date)
);

CREATE INDEX idx_flight_instances_route ON flight_instances(route_id);
CREATE INDEX idx_flight_instances_departure ON flight_instances(departure_date);
CREATE INDEX idx_flight_instances_schedule ON flight_instances(schedule_id);
CREATE INDEX idx_flight_instances_search ON flight_instances(route_id, departure_date);

-- =============================================
-- Seat Map
-- =============================================

CREATE TABLE IF NOT EXISTS seats (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aircraft_id         UUID NOT NULL REFERENCES aircraft(id),
    seat_number         VARCHAR(10) NOT NULL,
    cabin_class_id      UUID NOT NULL REFERENCES cabin_classes(id),
    seat_type           VARCHAR(20) NOT NULL DEFAULT 'standard'
                        CHECK (seat_type IN ('standard', 'window', 'middle', 'aisle', 'exit_row')),
    row_number          INTEGER NOT NULL,
    column_letter       VARCHAR(1) NOT NULL,
    is_exit_row         BOOLEAN NOT NULL DEFAULT FALSE,
    is_active           BOOLEAN NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (aircraft_id, seat_number)
);

CREATE INDEX idx_seats_aircraft ON seats(aircraft_id);
CREATE INDEX idx_seats_cabin ON seats(cabin_class_id);

CREATE TABLE IF NOT EXISTS flight_instance_fare_classes (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    flight_instance_id  UUID NOT NULL REFERENCES flight_instances(id) ON DELETE CASCADE,
    fare_class_id       UUID NOT NULL REFERENCES fare_classes(id),
    price               DECIMAL(10,2) NOT NULL,
    available_count     INTEGER NOT NULL DEFAULT 0,
    total_count         INTEGER NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (flight_instance_id, fare_class_id)
);

CREATE INDEX idx_fic_flight ON flight_instance_fare_classes(flight_instance_id);

-- =============================================
-- Users
-- =============================================

CREATE TABLE IF NOT EXISTS users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           VARCHAR(255) NOT NULL UNIQUE,
    password_hash   VARCHAR(255),
    name            VARCHAR(255) NOT NULL,
    role            VARCHAR(20) NOT NULL DEFAULT 'user'
                    CHECK (role IN ('user', 'admin', 'agent')),
    phone           VARCHAR(20),
    avatar_url      TEXT,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    last_login_at   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_role ON users(role);

-- =============================================
-- Bookings
-- =============================================

CREATE TABLE IF NOT EXISTS pnrs (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code                VARCHAR(8) NOT NULL UNIQUE,
    user_id             UUID REFERENCES users(id),
    status              VARCHAR(20) NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending', 'confirmed', 'cancelled', 'completed')),
    passenger_count     INTEGER NOT NULL DEFAULT 1,
    total_amount        DECIMAL(10,2) NOT NULL DEFAULT 0,
    currency            VARCHAR(3) NOT NULL DEFAULT 'USD',
    contact_email       VARCHAR(255) NOT NULL,
    contact_phone       VARCHAR(20),
    payment_intent_id   VARCHAR(255),
    payment_status      VARCHAR(20) NOT NULL DEFAULT 'pending'
                        CHECK (payment_status IN ('pending', 'processing', 'completed', 'refunded', 'failed')),
    idempotency_key     VARCHAR(255) UNIQUE,
    expires_at          TIMESTAMPTZ,
    confirmed_at        TIMESTAMPTZ,
    cancelled_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_pnrs_code ON pnrs(code);
CREATE INDEX idx_pnrs_user ON pnrs(user_id);
CREATE INDEX idx_pnrs_status ON pnrs(status);
CREATE INDEX idx_pnrs_idempotency ON pnrs(idempotency_key);
CREATE INDEX idx_pnrs_created ON pnrs(created_at);

CREATE TABLE IF NOT EXISTS bookings (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pnr_id              UUID NOT NULL REFERENCES pnrs(id) ON DELETE CASCADE,
    flight_instance_id  UUID NOT NULL REFERENCES flight_instances(id),
    fare_class_id       UUID NOT NULL REFERENCES fare_classes(id),
    seat_id             UUID REFERENCES seats(id),
    status              VARCHAR(20) NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending', 'confirmed', 'checked_in', 'cancelled', 'no_show')),
    price               DECIMAL(10,2) NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_bookings_pnr ON bookings(pnr_id);
CREATE INDEX idx_bookings_flight ON bookings(flight_instance_id);
CREATE INDEX idx_bookings_seat ON bookings(seat_id);
CREATE INDEX idx_bookings_status ON bookings(status);

CREATE TABLE IF NOT EXISTS passengers (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id          UUID NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
    first_name          VARCHAR(100) NOT NULL,
    last_name          VARCHAR(100) NOT NULL,
    date_of_birth       DATE,
    passport_number     VARCHAR(50),
    nationality         VARCHAR(3),
    email               VARCHAR(255),
    phone               VARCHAR(20),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_passengers_booking ON passengers(booking_id);

-- =============================================
-- Check-in
-- =============================================

CREATE TABLE IF NOT EXISTS check_ins (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id          UUID NOT NULL UNIQUE REFERENCES bookings(id),
    checked_in_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    seat_id             UUID REFERENCES seats(id),
    boarding_group      VARCHAR(2),
    boarding_number     INTEGER,
    gate                VARCHAR(10),
    boarding_pass_data  JSONB,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_checkins_booking ON check_ins(booking_id);

-- =============================================
-- Pricing Rules
-- =============================================

CREATE TABLE IF NOT EXISTS pricing_rules (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(255) NOT NULL,
    rule_type       VARCHAR(50) NOT NULL
                    CHECK (rule_type IN ('percentage', 'fixed', 'override')),
    value           DECIMAL(10,2) NOT NULL,
    fare_class_id   UUID REFERENCES fare_classes(id),
    route_id        UUID REFERENCES routes(id),
    flight_instance_id UUID REFERENCES flight_instances(id),
    active          BOOLEAN NOT NULL DEFAULT TRUE,
    priority        INTEGER NOT NULL DEFAULT 0,
    valid_from      TIMESTAMPTZ,
    valid_until     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_pricing_rules_active ON pricing_rules(active);
CREATE INDEX idx_pricing_rules_fare ON pricing_rules(fare_class_id);

-- =============================================
-- Reports / Audit
-- =============================================

CREATE TABLE IF NOT EXISTS audit_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID REFERENCES users(id),
    action          VARCHAR(100) NOT NULL,
    entity_type     VARCHAR(50) NOT NULL,
    entity_id       UUID,
    details         JSONB,
    ip_address      INET,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_user ON audit_logs(user_id);
CREATE INDEX idx_audit_entity ON audit_logs(entity_type, entity_id);
CREATE INDEX idx_audit_created ON audit_logs(created_at);

-- +migrate Down

DROP TABLE IF EXISTS audit_logs CASCADE;
DROP TABLE IF EXISTS pricing_rules CASCADE;
DROP TABLE IF EXISTS check_ins CASCADE;
DROP TABLE IF EXISTS passengers CASCADE;
DROP TABLE IF EXISTS bookings CASCADE;
DROP TABLE IF EXISTS pnrs CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS flight_instance_fare_classes CASCADE;
DROP TABLE IF EXISTS seats CASCADE;
DROP TABLE IF EXISTS flight_instances CASCADE;
DROP TABLE IF EXISTS flight_schedules CASCADE;
DROP TABLE IF EXISTS routes CASCADE;
DROP TABLE IF EXISTS fare_classes CASCADE;
DROP TABLE IF EXISTS cabin_classes CASCADE;
DROP TABLE IF EXISTS aircraft CASCADE;
DROP TABLE IF EXISTS airlines CASCADE;
DROP TABLE IF EXISTS airports CASCADE;
