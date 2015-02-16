
function parseError(jqXHR) {
	if (jqXHR.status == 0) {
		return {message: "network request failed"};
	}
	var err;
	try {
		err = JSON.parse(jqXHR.responseText).error;
	} catch(e) {
		if (jqXHR.status == 403) {
			err = {message: "Access denied"};
		} else {
			console.error(jqXHR.responseText);
			err = {message: "Unknown error"};
		}
	}
	return err;
}

module.exports = {
	// cb is function(success: bool, data: object, error: {message: string}, jqXHR: jqXHR)
	ajax: function(params, cb, async) {
		params.success = function(data) {
			cb(true, data, "", null);
		}
		params.error = function(jqXHR) {
			cb(false, null, parseError(jqXHR), jqXHR);
		}
		params.async = (async == true || async == null)
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
	updateMedicalLicenses: function(doctorID, licenses, cb) {
		this.ajax({
			type: "PUT",
			contentType: "application/json",
			url: "/doctors/" + doctorID + "/licenses",
			data: JSON.stringify({"licenses": licenses}),
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
	doctorProfileImageURL: function(id, pImageType) {
		return "/admin/api/doctors/" + encodeURIComponent(id) + "/profile_image/" + encodeURIComponent(pImageType);
	},
	updateDoctorProfileImage: function(id, pImageType, formData, cb) {
		this.ajax({
			type: "PUT",
			cache: false,
			contentType: false,
			processData: false,
			url: "/doctors/" + encodeURIComponent(id) + "/profile_image/" + encodeURIComponent(pImageType),
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
	updateResourceGuide: function(id, guideUpdate, cb) {
		this.ajax({
			type: "PATCH",
			contentType: "application/json",
			url: "/guides/resources/" + encodeURIComponent(id),
			data: JSON.stringify(guideUpdate),
			dataType: "json"
		}, cb);
	},
	createResourceGuide: function(guide, cb) {
		this.ajax({
			type: "POST",
			contentType: "application/json",
			url: "/guides/resources",
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
	},

	// Librato

	libratoQueryComposite: function(compose, resolution, start_time, end_time, count, cb) {
		var query = "compose=" + encodeURIComponent(compose);
		query += "&resolution=" + encodeURIComponent(resolution);
		query += "&start_time=" + encodeURIComponent(start_time);
		if (end_time) {
			query += "&end_time=" + encodeURIComponent(end_time);
		}
		if (count) {
			query += "&count=" + encodeURIComponent(count);
		}
		this.ajax({
			type: "GET",
			url: "/librato/composite?" + query,
			dataType: "json"
		}, cb);
	},

	// Stripe

	stripeCharges: function(limit, cb) {
		var query = "";
		if (limit) {
			query += "limit=" + encodeURIComponent(limit);
		}
		this.ajax({
			type: "GET",
			url: "/stripe/charges?" + query,
			dataType: "json"
		}, cb);
	},

	// Pathways

	pathway: function(id, cb) {
		this.ajax({
			type: "GET",
			url: "/pathways/" + encodeURIComponent(id),
			dataType: "json"
		}, cb);
	},
	pathways: function(activeOnly, cb) {
		var query = "";
		if (activeOnly) {
			query += "active_only=true";
		}
		this.ajax({
			type: "GET",
			url: "/pathways?" + query,
			dataType: "json"
		}, cb);
	},
	pathwayMenu: function(cb) {
		this.ajax({
			type: "GET",
			url: "/pathways/menu",
			dataType: "json"
		}, cb);
	},
	createPathway: function(pathway, cb) {
		this.ajax({
			type: "POST",
			contentType: "application/json",
			url: "/pathways",
			data: JSON.stringify({pathway: pathway}),
			dataType: "json"
		}, cb);
	},
	updatePathway: function(id, details, cb) {
		this.ajax({
			type: "PUT",
			contentType: "application/json",
			url: "/pathways/" + encodeURIComponent(id),
			data: JSON.stringify({details: details}),
			dataType: "json"
		}, cb);
	},
	updatePathwayMenu: function(menu, cb) {
		this.ajax({
			type: "PUT",
			contentType: "application/json",
			url: "/pathways/menu",
			data: JSON.stringify(menu),
			dataType: "json"
		}, cb);
	},

	// Templates
	layoutVersions: function(cb) {
		this.ajax({
			type: "GET",
			url: "/layouts/version",
			dataType: "json"
		}, cb);
	},
	question: function(tag, language_id, version, cb, async) {
		version = version == null ? 1 : version
		query = "tag="+tag+"&version="+version+"&language_id="+language_id
		this.ajax({
			type: "GET",
			url: "/layouts/versioned_question?" + query,
			dataType: "json"
		}, cb, async);
	},
	template: function(pathway_tag, purpose, major, minor, patch, cb) {
		query = "pathway_tag="+pathway_tag+"&purpose="+purpose+"&major="+major+"&minor="+minor+"&patch="+patch
		this.ajax({
			type: "GET",
			url: "/layouts/template?" + query,
			dataType: "json"
		}, cb);
	},
	layoutUpload: function(formData, cb, async) {
		this.ajax({
			type: "POST",
			cache: false,
			contentType: false,
			processData: false,
			url: "/layout",
			data: formData
		}, cb, async);
	},
	submitQuestion: function(question, cb, async) {
		this.ajax({
			type: "POST",
			contentType: "application/json",
			url: "/layouts/versioned_question",
			data: JSON.stringify(question),
			dataType: "json"
		}, cb, async);
	},

	// STP
	sampleTreatmentPlan: function(pathway, cb) {
		query = "pathway_tag="+pathway
		this.ajax({
			type: "GET",
			url: "/sample_treatment_plan?" + query,
			dataType: "json"
		}, cb);
	},
	updateSampleTreatmentPlan: function(pathway, stp, cb) {
		this.ajax({
			type: "PUT",
			contentType: "application/json",
			url: "/sample_treatment_plan",
			data: JSON.stringify({pathway_tag: pathway, sample_treatment_plan: stp}),
			dataType: "json"
		}, cb);
	},
};
