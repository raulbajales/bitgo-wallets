-- 001_initial_schema.sql
-- Create extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    role VARCHAR(50) NOT NULL DEFAULT 'end_user', -- end_user, operator, approver, admin
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create organizations table (single org for now)
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create wallets table
CREATE TABLE wallets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    bitgo_wallet_id VARCHAR(255) UNIQUE NOT NULL, -- BitGo's internal wallet ID
    label VARCHAR(255) NOT NULL,
    coin VARCHAR(10) NOT NULL, -- e.g., 'btc', 'eth', 'tbtc4'
    wallet_type VARCHAR(10) NOT NULL CHECK (wallet_type IN ('warm', 'cold')),
    balance_string VARCHAR(50) DEFAULT '0', -- BitGo balance as string
    confirmed_balance_string VARCHAR(50) DEFAULT '0',
    spendable_balance_string VARCHAR(50) DEFAULT '0',
    is_active BOOLEAN DEFAULT true,
    frozen BOOLEAN DEFAULT false,
    
    -- BitGo specific fields
    multisig_type VARCHAR(20), -- e.g., 'tss', 'multisig'
    threshold INTEGER DEFAULT 2, -- for multisig (typically 2 of 3)
    
    -- Additional metadata
    tags TEXT[], -- array of tags
    metadata JSONB DEFAULT '{}', -- flexible metadata storage
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create wallet_memberships table (users associated with wallets)
CREATE TABLE wallet_memberships (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    wallet_id UUID NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL DEFAULT 'viewer', -- viewer, spender, admin
    permissions JSONB DEFAULT '{}', -- flexible permissions
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(wallet_id, user_id)
);

-- Create transfer_requests table
CREATE TABLE transfer_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    wallet_id UUID NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
    requested_by_user_id UUID NOT NULL REFERENCES users(id),
    
    -- Transfer details
    recipient_address VARCHAR(255) NOT NULL,
    amount_string VARCHAR(50) NOT NULL, -- Amount as string to avoid precision issues
    coin VARCHAR(10) NOT NULL,
    
    -- Transfer type specific
    transfer_type VARCHAR(10) NOT NULL CHECK (transfer_type IN ('warm', 'cold')),
    
    -- Status tracking
    status VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (status IN (
        'draft',           -- Initial creation
        'submitted',       -- Submitted to BitGo
        'pending_approval',-- Awaiting approvals
        'approved',        -- All approvals received
        'signed',          -- Transaction signed
        'broadcast',       -- Transaction broadcast to network
        'confirmed',       -- Transaction confirmed
        'completed',       -- Transfer completed successfully
        'failed',          -- Transfer failed
        'rejected',        -- Transfer rejected
        'cancelled'        -- Transfer cancelled
    )),
    
    -- BitGo integration
    bitgo_transfer_id VARCHAR(255) UNIQUE, -- BitGo's transfer ID
    bitgo_txid VARCHAR(255), -- BitGo's transaction ID
    transaction_hash VARCHAR(255), -- Blockchain transaction hash
    
    -- Fee information
    fee VARCHAR(50), -- Transaction fee
    fee_rate VARCHAR(50), -- Fee rate
    
    -- Approval tracking
    required_approvals INTEGER DEFAULT 1,
    received_approvals INTEGER DEFAULT 0,
    
    -- Additional data
    memo TEXT, -- Optional memo/note
    fee_string VARCHAR(50), -- Transaction fee
    estimated_fee_string VARCHAR(50), -- Estimated fee
    
    -- Timestamps
    submitted_at TIMESTAMP WITH TIME ZONE,
    approved_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    failed_at TIMESTAMP WITH TIME ZONE,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create audit_logs table
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Context
    user_id UUID REFERENCES users(id),
    organization_id UUID REFERENCES organizations(id),
    wallet_id UUID REFERENCES wallets(id),
    transfer_request_id UUID REFERENCES transfer_requests(id),
    
    -- Action details
    action VARCHAR(100) NOT NULL, -- e.g., 'wallet_created', 'transfer_submitted'
    resource_type VARCHAR(50) NOT NULL, -- e.g., 'wallet', 'transfer_request'
    resource_id VARCHAR(255), -- Generic resource ID
    
    -- Audit trail
    old_values JSONB,
    new_values JSONB,
    metadata JSONB DEFAULT '{}',
    
    -- Request context
    ip_address INET,
    user_agent TEXT,
    correlation_id UUID, -- For tracing across services
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX idx_wallets_bitgo_id ON wallets(bitgo_wallet_id);
CREATE INDEX idx_wallets_org_id ON wallets(organization_id);
CREATE INDEX idx_wallets_type ON wallets(wallet_type);
CREATE INDEX idx_wallets_coin ON wallets(coin);

CREATE INDEX idx_wallet_memberships_wallet ON wallet_memberships(wallet_id);
CREATE INDEX idx_wallet_memberships_user ON wallet_memberships(user_id);

CREATE INDEX idx_transfer_requests_wallet ON transfer_requests(wallet_id);
CREATE INDEX idx_transfer_requests_user ON transfer_requests(requested_by_user_id);
CREATE INDEX idx_transfer_requests_status ON transfer_requests(status);
CREATE INDEX idx_transfer_requests_type ON transfer_requests(transfer_type);
CREATE INDEX idx_transfer_requests_created ON transfer_requests(created_at);

CREATE INDEX idx_audit_logs_user ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX idx_audit_logs_created ON audit_logs(created_at);
CREATE INDEX idx_audit_logs_correlation ON audit_logs(correlation_id);

-- Insert default organization
INSERT INTO organizations (id, name, description) 
VALUES (
    uuid_generate_v4(),
    'BitGo Wallets Demo',
    'Default organization for BitGo Wallets demo application'
);

-- Insert default admin user
INSERT INTO users (id, email, password_hash, first_name, last_name, role) 
VALUES (
    uuid_generate_v4(),
    'admin@bitgo.com',
    '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', -- 'password' hashed
    'Admin',
    'User',
    'admin'
);