
module.exports = {
	AnalyticsReportsView: "analytics_reports.view",
	AnalyticsReportsEdit: "analytics_reports.edit",
	AdminAccountsView: "admin_accounts.view",
	AdminAccountsEdit: "admin_accounts.edit",
	DoctorsView: "doctors.view",
	DoctorsEdit: "doctors.edit",
	EmailView: "email.view",
	EmailEdit: "email.edit",

	has: function(perm) {
		if (typeof perm != "string") {
			console.error("Perms.has expected a 'string' not '" + typeof perm + "'")
		}
		return Spruce.AccountPermissions[perm] || false;
	}
};
