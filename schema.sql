-- Schema do banco (Neon / PostgreSQL).
-- O backend aplica este schema automaticamente na inicialização,
-- mas você também pode executá-lo manualmente no SQL Editor do Neon.

CREATE TABLE IF NOT EXISTS pesagens (
    id        BIGSERIAL PRIMARY KEY,
    brinco    TEXT NOT NULL DEFAULT '',          -- identificação do animal (opcional)
    peso_kg   NUMERIC(7,2) NOT NULL CHECK (peso_kg > 0),
    data_hora TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_pesagens_data_hora ON pesagens (data_hora);
