-- NOTE: The need for this migration has arisen due to the fact that we changed the stripe key in production from the live key to 
-- a test key. Consequently, patients are unable to log in to their accounts as the call to get credit cards is failing, because the client
-- app intentionally waits to get the credit cards so as to know where in the flow of a visit the user stands.
-- The reason for a switch to the test key was to get around the problem of users having to enter a real credit card when we aren't charging for the 
-- product yet. Getting rid of the payment information will just mean that hte user has to re-enter it for those that have not submitted a card yet.

-- Add the customer_id column to capture which token belongs to which customer in case we need to bring it back up 
alter table credit_card add column payment_service_customer_id varchar(500);

-- Update the customer_id information with the values from the patient table 
update credit_card inner join patient on patient.id = credit_card.patient_id set credit_card.payment_service_customer_id = patient.payment_service_customer_id;

-- Mark all credit cards with a status that we can use to retreive the cards in the event that things go wrong
update credit_card set status='INACTIVE_LIVE_KEY_ISSUE';

-- Now that we have the mapping between the credit card tokens and the customer id, 
-- nullify the customer id from the patient table so that the call to get credit cards doesn't fail
update patient set payment_service_customer_id = NULL;