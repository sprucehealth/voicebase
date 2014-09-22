-- alter table scheduled_message_template drop foreign key scheduled_message_template_ibfk_1;
-- alter table scheduled_message_template modify column creator_account_id int unsigned;
-- alter table scheduled_message_template add foreign key (creator_account_id) references account(id);

insert into scheduled_message_template (name, event, schedule_period, message)
	values ('Care coordinator message for insured patient', 'insured_patient', 120, 
		'Hi {{.PatientFirstName}},\n\nMy name is Holly, and I\'m your Care Coordinator. I wanted to send you a quick note to say hello, and let you know I\'m here to answer any questions you have and help you have a great experience on Spruce.\n\nOne way I can help is to check whether the medications your dermatologist prescribes are covered by your insurance, so there aren\'t any surprises at the pharmacy counter. If you want me to do that, just send a photo of the front and back of your insurance card. Or, you can type out:\n(1) Name of the insurance company (e.g., Anthem Blue Cross)\n(2) Name of your plan (e.g., Premier HMO)\n(3) Member number / BIN\n(4) the Customer Service phone number on the back of your card\n\nWhether or not you decide to send your info, I\'d recommend calling the pharmacy before you pick up any prescriptions in your treatment plan. That way you\'ll know if they\'re ready, how much they\'ll cost, and can message me if you have any problems.\n\nWarmly,\n{{.ProviderShortDisplayName}}');

insert into scheduled_message_template (name, event, schedule_period, message)
	values ('Care coordinator message for uninsured patient', 'uninsured_patient', 120, 
		'Hi {{.PatientFirstName}},\n\nMy name is Holly, and I\'m your Care Coordinator. I wanted to send you a quick note to say hello, and let you know I\'m here to answer any questions you have and help you have a great experience on Spruce.\n\nOne thing I\'ve been recommending to all patients, particularly if you are uninsured, is calling the pharmacy before you pick up any prescriptions in your treatment plan. That way you\'ll know if they\'re ready, how much they\'ll cost, and can message me if you have any problems.\n\nWarmly,\n{{.ProviderShortDisplayName}}');
