
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    role TEXT NOT NULL, -- Ex. 'admin', 'employee', 'visitor'
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE doors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    location TEXT NOT NULL,
    min_role_required TEXT NOT NULL
);

CREATE TABLE access_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    door_id UUID REFERENCES doors(id),
    access_time TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    granted BOOLEAN NOT NULL
);