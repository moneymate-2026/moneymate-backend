CREATE TABLE auth.role_permissions (
    role_id UUID NOT NULL,
    permission_id UUID NOT NULL,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (role_id, permission_id),

    FOREIGN KEY (role_id)
        REFERENCES auth.roles(id)
        ON DELETE CASCADE,

    FOREIGN KEY (permission_id)
        REFERENCES auth.permissions(id)
        ON DELETE CASCADE
);