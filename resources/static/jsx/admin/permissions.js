
module.exports = {
	AdminAccountsEdit: "admin_accounts.edit",
	AdminAccountsView: "admin_accounts.view",
	AnalyticsReportsEdit: "analytics_reports.edit",
	AnalyticsReportsView: "analytics_reports.view",
	DoctorsEdit: "doctors.edit",
	DoctorsView: "doctors.view",
	EmailEdit: "email.edit",
	EmailView: "email.view",
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
	FinancialView: "financial.view",

	has: function(perm) {
		if (typeof perm != "string") {
			console.error("Perms.has expected a 'string' not '" + typeof perm + "'")
		}
		return Spruce.AccountPermissions[perm] || false;
	}
};
