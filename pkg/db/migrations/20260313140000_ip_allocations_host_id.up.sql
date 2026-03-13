-- Add host_id to ip_allocations for linking IP allocations to hosts.
ALTER TABLE ip_allocations ADD COLUMN host_id BIGINT REFERENCES hosts(id) ON DELETE SET NULL;
CREATE INDEX idx_ip_allocations_host_id ON ip_allocations(host_id);
