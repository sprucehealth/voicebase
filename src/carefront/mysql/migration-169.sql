alter table rx_refill_request add column erx_id int unsigned;
update deny_refill_reason set reason_code='DeniedPatientUnknown' where reason_code='AA';
update deny_refill_reason set reason_code='DeniedPatientNotUnderCare' where reason_code='AB';
update deny_refill_reason set reason_code='DeniedPatientNoLongerUnderPatientCare' where reason_code='AC';
update deny_refill_reason set reason_code='DeniedTooSoon' where reason_code='AD';
update deny_refill_reason set reason_code='DeniedNeverPrescribed' where reason_code='AE';
update deny_refill_reason set reason_code='DeniedHavePatientContact' where reason_code='AF';
update deny_refill_reason set reason_code='DeniedRefillInappropriate' where reason_code='AG';
update deny_refill_reason set reason_code='DeniedAlreadyPickedUp' where reason_code='AH';
update deny_refill_reason set reason_code='DeniedAlreadyPickedUpPartialFill' where reason_code='AJ';
update deny_refill_reason set reason_code='DeniedNotPickedUp' where reason_code='AK';
update deny_refill_reason set reason_code='DeniedChangeInappropriate' where reason_code='AL';
update deny_refill_reason set reason_code='DeniedNeedAppointment' where reason_code='AM';
update deny_refill_reason set reason_code='DeniedPrescriberNotAssociateWithLocation' where reason_code='AN';
update deny_refill_reason set reason_code='DeniedNoPriorAuthAttempt' where reason_code='AO';
update deny_refill_reason set reason_code='DeniedAlreadyHandled' where reason_code='AP';
insert into deny_refill_reason (reason_code, reason) values ('DeniedNewRx', 'New RX to follow');
	

