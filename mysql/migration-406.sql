-- Updating the scheduled message for both uninsured and insured cases with new content
UPDATE scheduled_message_template 
SET message = 'Hi {{.PatientFirstName}},

My name is {{.ProviderFirstName}}, and I''m your Care Coordinator. I wanted to send you a quick note to say hello, and let you know I''m here to answer any questions you have and help you have a great experience on Spruce.

One way I can help is to check whether the medications your dermatologist prescribes are covered by your insurance, so there aren''t any surprises at the pharmacy counter. If you want me to do that, just send a photo of the front and back of your insurance card. Or, you can type out:
(1) Name of the insurance company (e.g., Anthem Blue Cross)
(2) Name of your plan (e.g., Premier HMO)
(3) Member number, Group number, Rx Bin, and PCN number
(4) the Customer Service phone number on the back of your card

Whether you decided to send your info, please do not go to the pharmacy without either calling them first, or hearing from me about medication pricing. In some cases I may be able to provide you with discounts that will be lower than what the pharmacy quotes you directly.

Warmly,
{{.ProviderShortDisplayName}}'
where event='insured_patient';


UPDATE scheduled_message_template
SET message = 'Hi {{.PatientFirstName}},

My name is {{.ProviderFirstName}}, and I''m your Care Coordinator. I wanted to send you a quick note to say hello, and let you know I''m here to answer any questions you have and help you have a great experience on Spruce.

In some cases I may be able to provide you with discounts that will be lower than what the pharmacy quotes you directly, particularly if you are uninsured.  Please do not go to the pharmacy without either calling them first, or hearing from me about medication pricing.

Warmly,
{{.ProviderShortDisplayName}}'
where event = 'uninsured_patient';

