/* @flow */

module.exports = {
	AdminAccountsEdit: "admin_accounts.edit",
	AdminAccountsView: "admin_accounts.view",
	AnalyticsReportsEdit: "analytics_reports.edit",
	AnalyticsReportsView: "analytics_reports.view",
	AppMessageTemplatesEdit: "sched_msgs.edit",
	AppMessageTemplatesView: "sched_msgs.view",
	CaseView: "case.view",
	CareCoordinatorView: "care_coordinator.view",
	CareCoordinatorEdit: "care_coordinator.edit",
	CfgEdit: "cfg.edit",
	CfgView: "cfg.view",
	DoctorsEdit: "doctors.edit",
	DoctorsView: "doctors.view",
	EmailEdit: "email.edit",
	EmailView: "email.view",
	FinancialView: "financial.view",
	FTPEdit: "ftp.edit",
	FTPView: "ftp.view",
	LayoutEdit: "layout.edit",
	LayoutView: "layout.view",
	PathwaysEdit: "pathways.edit",
	PathwaysView: "pathways.view",
	ResourceGuidesEdit: "resource_guides.edit",
	ResourceGuidesView: "resource_guides.view",
	RXGuidesEdit: "rx_guides.edit",
	RXGuidesView: "rx_guides.view",
	STPEdit: "stp.edit",
	STPView: "stp.view",

	has: function(perm: string): bool {
		return Spruce.AccountPermissions[perm] || false;
	}
};
