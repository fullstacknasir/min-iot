CREATE TABLE IF NOT EXISTS custom_widgets (
    id VARCHAR(64) PRIMARY KEY,
    organization_id VARCHAR(64) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    widget_type VARCHAR(64) NOT NULL DEFAULT 'telemetry',
    template_html TEXT NOT NULL DEFAULT '',
    style_css TEXT NOT NULL DEFAULT '',
    controller_js TEXT NOT NULL DEFAULT '',
    settings_schema JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_custom_widgets_org ON custom_widgets (organization_id);
CREATE INDEX IF NOT EXISTS idx_custom_widgets_updated ON custom_widgets (organization_id, updated_at DESC);
