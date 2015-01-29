
module.exports = {
	AdminAccountsEdit: "admin_accounts.edit",
	AdminAccountsView: "admin_accounts.view",
	AnalyticsReportsEdit: "analytics_reports.edit",
	AnalyticsReportsView: "analytics_reports.view",
	DoctorsEdit: "doctors.edit",
	DoctorsView: "doctors.view",
	EmailEdit: "email.edit",
	EmailView: "email.view",
	PathwaysEdit: "pathways.edit",
	PathwaysView: "pathways.view",
	LayoutEdit: "layout.edit",
	LayoutView: "layout.view",

	has: function(perm) {
		if (typeof perm != "string") {
			console.error("Perms.has expected a 'string' not '" + typeof perm + "'")
		}
		return Spruce.AccountPermissions[perm] || false;
	}
};
