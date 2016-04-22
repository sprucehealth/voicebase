-- Drop this unique index for the (provisioned_for, endpoint_type) pair because
-- now that you can deprovision endpoints it is possible for an entity
-- to create the same endpoint type again.
ALTER TABLE provisioned_endpoint DROP INDEX provisioned_for;
