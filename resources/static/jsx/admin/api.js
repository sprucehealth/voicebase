
function parseError(jqXHR) {
	if (jqXHR.status == 0) {
		return {message: "network request failed"};
	}
	var err;
	try {
		err = JSON.parse(jqXHR.responseText).error;
	} catch(e) {
		console.error(e);
		console.error(jqXHR.responseText);
		err = {message: "Unknown error"};
	}
	return err;
}

module.exports = {
	// cb is function(success: bool, data: object, error: {message: string}, jqXHR: jqXHR)
	ajax: function(params, cb) {
		params.success = function(data) {
			cb(true, data, "", null);
		}
		params.error = function(jqXHR) {
			cb(false, null, parseError(jqXHR), jqXHR);
		}
		params.url = "/admin/api" + params.url;
		jQuery.ajax(params);
	},

	// Accounts

	account: function(id, cb) {
		this.ajax({
			type: "GET",
			url: "/accounts/" + encodeURIComponent(id),
			dataType: "json"
		}, cb);
	},
	accountPhoneNumbers: function(id, cb) {
		this.ajax({
			type: "GET",
			url: "/accounts/" + encodeURIComponent(id) + "/phones",
			dataType: "json"
		}, cb);
	},
	updateAccount: function(id, account, cb) {
		this.ajax({
			type: "PATCH",
			contentType: "application/json",
			url: "/accounts/" + id,
			data: JSON.stringify(account),
			dataType: "json"
		}, cb);
	},

	// Doctors / care providers

	doctor: function(id, cb) {
		this.ajax({
			type: "GET",
			url: "/doctors/" + encodeURIComponent(id),
			dataType: "json"
		}, cb);
	},
	doctorAttributes: function(id, cb) {
		this.ajax({
			type: "GET",
			url: "/doctors/" + encodeURIComponent(id) + "/attributes",
			dataType: "json"
		}, cb);
	},
	medicalLicenses: function(doctorID, cb) {
		this.ajax({
			type: "GET",
			url: "/doctors/" + doctorID + "/licenses",
			dataType: "json"
		}, cb);
	},
	searchDoctors: function(query, cb) {
		this.ajax({
			type: "GET",
			url: "/doctors?q=" + encodeURIComponent(query),
			dataType: "json"
		}, cb);
	},
	careProviderProfile: function(doctorID, cb) {
		this.ajax({
			type: "GET",
			url: "/doctors/" + doctorID + "/profile",
			dataType: "json"
		}, cb);
	},
	updateCareProviderProfile: function(doctorID, profile, cb) {
		this.ajax({
			type: "PUT",
			contentType: "application/json",
			url: "/doctors/" + doctorID + "/profile",
			data: JSON.stringify(profile),
			dataType: "json"
		}, cb);
	},
	doctorOnboarding: function(cb) {
		this.ajax({
			type: "GET",
			url: "/dronboarding",
			dataType: "json"
		}, cb);
	},
	doctorThumbnailURL: function(id, size) {
		return "/admin/api/doctors/" + encodeURIComponent(id) + "/thumbnail/" + encodeURIComponent(size);
	},
	updateDoctorThumbnail: function(id, size, formData, cb) {
		this.ajax({
			type: "PUT",
			cache: false,
			contentType: false,
			processData: false,
			url: "/doctors/" + encodeURIComponent(id) + "/thumbnail/" + encodeURIComponent(size),
			data: formData
		}, cb);
	},

	// Guides

	resourceGuide: function(id, cb) {
		this.ajax({
			type: "GET",
			url: "/guides/resources/" + id,
			dataType: "json"
		}, cb);
	},
	resourceGuidesList: function(withLayouts, sectionsOnly, cb) {
		var params = [];
		if (withLayouts) {
			params.push("with_layouts=1")
		}
		if (sectionsOnly) {
			params.push("sections_only=1")
		}
		this.ajax({
			type: "GET",
			url: "/guides/resources?" + params.join("&"),
			dataType: "json"
		}, cb);
	},
	resourceGuidesImport: function(formData, cb) {
		this.ajax({
			type: "PUT",
			cache: false,
			contentType: false,
			processData: false,
			url: "/guides/resources",
			data: formData
		}, cb);
	},
	resourceGuidesExport: function(cb) {
		this.ajax({
			type: "GET",
			url: "/guides/resources?with_layouts=1&indented=1",
			dataType: "text"
		}, cb);
	},
	rxGuide: function(ndc, withHTML, cb) {
		var params = "";
		if (withHTML) {
			params = "?with_html=1"
		}
		this.ajax({
			type: "GET",
			url: "/guides/rx/" + ndc + params,
			dataType: "json"
		}, cb);
	},
	rxGuidesList: function(cb) {
		this.ajax({
			type: "GET",
			url: "/guides/rx",
			dataType: "json"
		}, cb);
	},
	rxGuidesImport: function(formData, cb) {
		this.ajax({
			type: "PUT",
			cache: false,
			contentType: false,
			processData: false,
			url: "/guides/rx",
			data: formData
		}, cb);
	},
	updateResourceGuide: function(id, guide, cb) {
		this.ajax({
			type: "POST",
			contentType: "application/json",
			url: "/guides/resources/" + id,
			data: JSON.stringify(guide),
			dataType: "json"
		}, cb);
	},

	// Analytics

	analyticsQuery: function(q, cb) {
		this.ajax({
			type: "POST",
			contentType: "application/json",
			url: "/analytics/query",
			data: JSON.stringify({query: q}),
			dataType: "json"
		}, cb);
	},
	listAnalyticsReports: function(cb) {
		this.ajax({
			type: "GET",
			url: "/analytics/reports",
			dataType: "json"
		}, cb);
	},
	analyticsReport: function(id, cb) {
		this.ajax({
			type: "GET",
			url: "/analytics/reports/" + encodeURIComponent(id),
			dataType: "json"
		}, cb);
	},
	runAnalyticsReport: function(id, cb) {
		this.ajax({
			type: "POST",
			contentType: "application/json",
			url: "/analytics/reports/" + encodeURIComponent(id) + "/run",
			dataType: "json"
		}, cb);
	},
	createAnalyticsReport: function(name, query, presentation, cb) {
		this.ajax({
			type: "POST",
			contentType: "application/json",
			url: "/analytics/reports",
			data: JSON.stringify({name: name, query: query, presentation: presentation}),
			dataType: "json"
		}, cb);
	},
	updateAnalyticsReport: function(id, name, query, presentation, cb) {
		this.ajax({
			type: "POST",
			contentType: "application/json",
			url: "/analytics/reports/" + encodeURIComponent(id),
			data: JSON.stringify({name: name, query: query, presentation: presentation}),
			dataType: "json"
		}, cb);
	},

	// Email

	listEmailTypes: function(cb) {
		this.ajax({
			type: "GET",
			url: "/email/types",
			dataType: "json"
		}, cb);
	},
	listEmailSenders: function(cb) {
		this.ajax({
			type: "GET",
			url: "/email/senders",
			dataType: "json"
		}, cb);
	},
	listEmailTemplates: function(typeKey, cb) {
		this.ajax({
			type: "GET",
			url: "/email/templates?type=" + encodeURIComponent(typeKey || ""),
			dataType: "json"
		}, cb);
	},
	createEmailTemplate: function(tmpl, cb) {
		this.ajax({
			type: "POST",
			contentType: "application/json",
			url: "/email/templates",
			data: JSON.stringify(tmpl),
			dataType: "json"
		}, cb);
	},
	updateEmailTemplate: function(tmpl, cb) {
		this.ajax({
			type: "PUT",
			contentType: "application/json",
			url: "/email/templates/" + encodeURIComponent(tmpl.id),
			data: JSON.stringify(tmpl),
			dataType: "json"
		}, cb);
	},
	testEmailTemplate: function(templateID, to, ctx, cb) {
		this.ajax({
			type: "POST",
			contentType: "application/json",
			url: "/email/templates/" + encodeURIComponent(templateID) + "/test",
			data: JSON.stringify({to: to, context: ctx}),
			dataType: "json"
		}, cb);
	},

	// Admin accounts

	searchAdmins: function(query, cb) {
		this.ajax({
			type: "GET",
			url: "/admins?q=" + encodeURIComponent(query),
			dataType: "json"
		}, cb);
	},
	adminAccount: function(id, cb) {
		this.ajax({
			type: "GET",
			url: "/admins/" + encodeURIComponent(id),
			dataType: "json"
		}, cb);
	},
	adminPermissions: function(id, cb) {
		this.ajax({
			type: "GET",
			url: "/admins/" + encodeURIComponent(id) + "/permissions",
			dataType: "json"
		}, cb);
	},
	adminGroups: function(id, cb) {
		this.ajax({
			type: "GET",
			url: "/admins/" + encodeURIComponent(id) + "/groups",
			dataType: "json"
		}, cb);
	},
	updateAdminGroups: function(id, groups, cb) {
		this.ajax({
			type: "POST",
			contentType: "application/json",
			url: "/admins/" + encodeURIComponent(id) + "/groups",
			data: JSON.stringify(groups),
			dataType: "json"
		}, cb);
	},
	availablePermissions: function(cb) {
		this.ajax({
			type: "GET",
			url: "/accounts/permissions",
			dataType: "json"
		}, cb);
	},
	availableGroups: function(withPermissions, cb) {
		var params = "";
		if (withPermissions) {
			params = "?with_perms=1";
		}
		this.ajax({
			type: "GET",
			url: "/accounts/groups" + params,
			dataType: "json"
		}, cb);
	},

	// Drugs

	searchDrugs: function(query, cb) {
		this.ajax({
			type: "GET",
			url: "/drugs?q=" + encodeURIComponent(query),
			dataType: "json"
		}, cb);
	}
};
