-- Updating the localized text to match that in the doctor review for the meidcation allergies list
start transaction;

update localized_text set ltext = 'Medication Allergies' where id=182;

commit;
