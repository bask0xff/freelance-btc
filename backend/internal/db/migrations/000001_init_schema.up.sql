CREATE OR REPLACE FUNCTION set_updated_at() RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE users (
    id             BIGSERIAL PRIMARY KEY,
    email          VARCHAR(255) NOT NULL UNIQUE,
    password_hash  VARCHAR(255) NOT NULL,
    role           VARCHAR(16) NOT NULL DEFAULT 'client' CHECK (role IN ('client', 'freelancer', 'admin')),
    display_name   VARCHAR(255) NOT NULL,
    payout_address VARCHAR(128),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE orders (
    id             BIGSERIAL PRIMARY KEY,
    client_id      BIGINT NOT NULL REFERENCES users(id),
    freelancer_id  BIGINT REFERENCES users(id),
    title          VARCHAR(255) NOT NULL,
    description    TEXT,
    amount_btc     NUMERIC(16, 8) NOT NULL,
    status         VARCHAR(20) NOT NULL DEFAULT 'open' CHECK (status IN (
                       'open', 'awaiting_payment', 'funded', 'in_progress', 'delivered',
                       'completed', 'disputed', 'refunded', 'cancelled'
                   )),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER orders_set_updated_at
    BEFORE UPDATE ON orders
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- 1 заказ = 1 депозитный адрес (label в bitcoind = 'order_' || order_id)
CREATE TABLE btc_addresses (
    id           BIGSERIAL PRIMARY KEY,
    order_id     BIGINT NOT NULL UNIQUE REFERENCES orders(id),
    address      VARCHAR(128) NOT NULL UNIQUE,
    address_type VARCHAR(16) NOT NULL DEFAULT 'bech32' CHECK (address_type IN ('bech32', 'p2sh-segwit', 'legacy')),
    label        VARCHAR(64) NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE btc_transactions (
    id            BIGSERIAL PRIMARY KEY,
    order_id      BIGINT NOT NULL REFERENCES orders(id),
    address       VARCHAR(128) NOT NULL,
    txid          VARCHAR(128) NOT NULL,
    amount_btc    NUMERIC(16, 8) NOT NULL,
    satoshi       BIGINT NOT NULL,
    confirmations INT NOT NULL DEFAULT 0,
    direction     VARCHAR(4) NOT NULL CHECK (direction IN ('in', 'out')),
    status        VARCHAR(16) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'confirmed')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uniq_tx UNIQUE (address, txid, direction)
);

CREATE TRIGGER btc_transactions_set_updated_at
    BEFORE UPDATE ON btc_transactions
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE disputes (
    id          BIGSERIAL PRIMARY KEY,
    order_id    BIGINT NOT NULL REFERENCES orders(id),
    opened_by   BIGINT NOT NULL REFERENCES users(id),
    reason      TEXT NOT NULL,
    status      VARCHAR(24) NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'resolved_client', 'resolved_freelancer', 'closed')),
    resolution  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ
);
