CREATE TABLE IF NOT EXISTS stock_deductions (
    request_id VARCHAR(100) PRIMARY KEY,
    payload_hash CHAR(64) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
