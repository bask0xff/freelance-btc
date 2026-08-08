-- Схема заменяет peewee-модели BtcAddress2 / BtcConfirmation из вашего скрипта
-- на структуру, привязанную к пользователям и заказам.

CREATE TABLE IF NOT EXISTS users (
    id            BIGINT AUTO_INCREMENT PRIMARY KEY,
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role          ENUM('client', 'freelancer', 'admin') NOT NULL DEFAULT 'client',
    display_name  VARCHAR(255) NOT NULL,
    payout_address VARCHAR(128) NULL, -- BTC-адрес фрилансера для вывода средств
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS orders (
    id             BIGINT AUTO_INCREMENT PRIMARY KEY,
    client_id      BIGINT NOT NULL,
    freelancer_id  BIGINT NULL,
    title          VARCHAR(255) NOT NULL,
    description    TEXT,
    amount_btc     DECIMAL(16,8) NOT NULL,
    status         ENUM(
                      'created',          -- заказ создан, ждём назначения фрилансера/адреса
                      'awaiting_payment', -- адрес выдан, ждём поступления BTC
                      'funded',           -- средства получены и подтверждены
                      'in_progress',      -- фрилансер работает
                      'delivered',        -- сдано, ждёт подтверждения клиента
                      'completed',        -- клиент принял, средства отправлены фрилансеру
                      'disputed',         -- спор
                      'refunded',         -- возврат клиенту
                      'cancelled'
                    ) NOT NULL DEFAULT 'created',
    created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (client_id) REFERENCES users(id),
    FOREIGN KEY (freelancer_id) REFERENCES users(id)
) ENGINE=InnoDB;

-- 1 заказ = 1 депозитный адрес (label в bitcoind = CONCAT('order_', order_id))
CREATE TABLE IF NOT EXISTS btc_addresses (
    id           BIGINT AUTO_INCREMENT PRIMARY KEY,
    order_id     BIGINT NOT NULL UNIQUE,
    address      VARCHAR(128) NOT NULL UNIQUE,
    address_type ENUM('bech32', 'p2sh-segwit', 'legacy') NOT NULL DEFAULT 'bech32',
    label        VARCHAR(64) NOT NULL,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (order_id) REFERENCES orders(id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS btc_transactions (
    id             BIGINT AUTO_INCREMENT PRIMARY KEY,
    order_id       BIGINT NOT NULL,
    address        VARCHAR(128) NOT NULL,
    txid           VARCHAR(128) NOT NULL,
    amount_btc     DECIMAL(16,8) NOT NULL,
    satoshi        BIGINT NOT NULL,
    confirmations  INT NOT NULL DEFAULT 0,
    direction      ENUM('in', 'out') NOT NULL,
    status         ENUM('pending', 'confirmed') NOT NULL DEFAULT 'pending',
    created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uniq_tx (address, txid, direction),
    FOREIGN KEY (order_id) REFERENCES orders(id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS disputes (
    id          BIGINT AUTO_INCREMENT PRIMARY KEY,
    order_id    BIGINT NOT NULL,
    opened_by   BIGINT NOT NULL,
    reason      TEXT NOT NULL,
    status      ENUM('open', 'resolved_client', 'resolved_freelancer', 'closed') NOT NULL DEFAULT 'open',
    resolution  TEXT,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    resolved_at DATETIME NULL,
    FOREIGN KEY (order_id) REFERENCES orders(id),
    FOREIGN KEY (opened_by) REFERENCES users(id)
) ENGINE=InnoDB;
