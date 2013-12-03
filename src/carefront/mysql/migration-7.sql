use carefront_db;
start transaction;

update localized_text set ltext = 'Medication Allergies' where id=99;
update localized_text set ltext = 'Current medications' where id=100;
update localized_text set ltext = 'Skin Conditions' where id=102;
update localized_text set ltext = 'Other Conditions' where id=103;
update localized_text set ltext = 'Pregnant/Nursing' where id=98;
update localized_text set ltext = 'Social History' where id=101;
update localized_text set ltext = 'Location of symptoms' where id=97;
update localized_text set ltext = 'Worsening symptoms' where id=92;
update localized_text set ltext = 'Type of treatments' where id=94;
update localized_text set ltext = 'OTC and Prescriptions tried' where id=95;
update localized_text set ltext = 'Recent changes making acne worse' where id=93;
commit;
