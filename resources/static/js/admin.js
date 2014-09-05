/** @jsx React.DOM */

var AdminAPI = {
	// cb is function(success: bool, data: object, jqXHR: jqXHR)
	ajax: function(params, cb) {
		params.success = function(data) {
			cb(true, data, null);
		}
		params.error = function(jqXHR) {
			cb(false, null, jqXHR);
		}
		params.url = "/admin/api" + params.url;
		jQuery.ajax(params);
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
	doctorSavedMessage: function(doctorID, cb) {
		this.ajax({
			type: "GET",
			url: "/doctors/" + doctorID + "/savedmessage",
			dataType: "json"
		}, cb);
	},
	updateDoctorSavedMessage: function(doctorID, msg, cb) {
		if (typeof msg != "string") {
			console.error("updateDoctorSavedMessage expected a string for msg instead of " + typeof msg);
			return
		}
		this.ajax({
			type: "PUT",
			contentType: "application/json",
			url: "/doctors/" + doctorID + "/savedmessage",
			data: JSON.stringify({"message": msg}),
			dataType: "json"
		}, cb);
	},
	doctorThumbnailURL: function(id, size) {
		return "/admin/api/doctors/" + encodeURIComponent(id) + "/thumbnail/" + encodeURIComponent(size);
	},
	updateDoctorThumbnail: function(id, size, formData, cb) {
		this.ajax({
			type: 'POST',
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
			type: 'PUT',
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
			type: 'PUT',
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
	}
};

var Perms = {
	AnalyticsReportsView: "analytics_reports.view",
	AnalyticsReportsEdit: "analytics_reports.edit",
	AdminAccountsView: "admin_accounts.view",
	AdminAccountsEdit: "admin_accounts.edit",

	has: function(perm) {
		if (typeof perm != "string") {
			console.error("Perms.has expected a 'string' not '" + typeof perm + "'")
		}
		return Spruce.AccountPermissions[perm] || false;
	}
};

function staticURL(path) {
	return Spruce.BaseStaticURL + path
}

var RouterNavigateMixin = {
	navigate: function(path) {
		if (path.indexOf(this.props.router.root) == 0) {
			path = path.substring(this.props.router.root.length, path.length);
		}
		this.props.router.navigate(path, {
			trigger : true
		});
	},
	onNavigate: function(e) {
		e.preventDefault();
		this.navigate(e.target.pathname);
		return false;
	}
};

var TopNav = React.createClass({displayName: "TopNav",
	mixins: [RouterNavigateMixin],
	render: function() {
		var leftMenuItems = this.props.leftItems.map(function(item) {
			var active = item.id == this.props.activeItem;
			return (
				<li key={item.id} className={active ? 'active' : ''}><a href={this.props.router.root + item.url} onClick={this.onNavigate}>{item.name}</a></li>
			);
		}.bind(this));
		var rightMenuItems = this.props.rightItems.map(function(item) {
			var active = item.id == this.props.activeItem;
			return (
				<li key={item.id} className={active ? 'active' : ''}><a href={this.props.router.root + item.url} onClick={this.onNavigate}>{item.name}</a></li>
			);
		}.bind(this));
		return (
			<div className="navbar navbar-inverse navbar-fixed-top" role="navigation">
				<div className="container-fluid">
					<div className="navbar-header">
						<button type="button" className="navbar-toggle" data-toggle="collapse" data-target=".navbar-collapse">
							<span className="sr-only">Toggle navigation</span>
							<span className="icon-bar"></span>
							<span className="icon-bar"></span>
							<span className="icon-bar"></span>
						</button>
						<a className="navbar-brand" href={this.props.router.root} onClick={this.onNavigate}>{this.props.name}</a>
					</div>
					<div className="collapse navbar-collapse">
						<ul className="nav navbar-nav">
							{leftMenuItems}
						</ul>
						<ul className="nav navbar-nav navbar-right">
							{rightMenuItems}
							<li><a href="/logout">Sign Out</a></li>
						</ul>
					</div>
				</div>
			</div>
		);
	}
});

var LeftNav = React.createClass({displayName: "LeftNav",
	mixins: [RouterNavigateMixin],
	render: function() {
		var navGroups = this.props.items.map(function(subItems, index) {
			return (
				<ul key={"leftNavGroup-"+index} className="nav nav-sidebar">
					{subItems.map(function(item) {
						var active = item.id == this.props.currentPage;
						return (
							<li key={item.id} className={active?"active":""}>
								<a href={item.url} onClick={this.onNavigate}>{item.name}</a>
							</li>
						);
					}.bind(this))}
				</ul>
			);
		}.bind(this));
		return (
			<div>
				<div className="row">
					<div className="col-sm-3 col-md-2 sidebar">
						{navGroups}
					</div>
				</div>
				<div className="col-sm-9 col-sm-offset-3 col-md-10 col-md-offset-2 main">
					{this.props.children}
				</div>
			</div>
		);
	}
});

var AdminRouter = Backbone.Router.extend({
	routes : {
		"": function() {
			this.current = "dashboard";
			this.params = {};
		},
		"doctors": function() {
			this.current = "doctorSearch";
			this.params = {};
		},
		"doctors/:doctorID/:page": function(doctorID, page) {
			this.current = "doctor";
			this.params = {doctorID: doctorID, page: page};
		},
		"guides/:page": function(page) {
			this.current = "guides";
			this.params = {page: page};
		},
		"guides/:page/:id": function(page, guideID) {
			this.current = "guides";
			this.params = {page: page, guideID: guideID};
		},
		"analytics/:page": function(page) {
			this.current = "analytics";
			this.params = {page: page};
		},
		"analytics/:page/:id": function(page, reportID) {
			this.current = "analytics";
			this.params = {page: page, reportID: reportID};
		},
		"accounts": function() {
			this.current = "accountsList";
			this.params = {};
		},
		"accounts/:accountID/:page": function(accountID, page) {
			this.current = "account";
			this.params = {accountID: accountID, page: page};
		}
	}
});

var Admin = React.createClass({displayName: "Admin",
	getDefaultProps: function() {
		var leftMenuItems = [
			{
				id: "dashboard",
				url: "",
				name: "Dashboard"
			},
			{
				id: "doctorSearch",
				url: "doctors",
				name: "Doctors"
			},
			{
				id: "guides",
				url: "guides/resources",
				name: "Guides"
			}
		];
		var rightMenuItems = [];

		if (Perms.has(Perms.AnalyticsReportsView)) {
			leftMenuItems.push({
				id: "analytics",
				url: "analytics/query",
				name: "Analytics"
			});
		}

		if (Perms.has(Perms.AdminAccountsView)) {
			leftMenuItems.push({
				id: "accounts",
				url: "accounts",
				name: "Accounts"
			});
		}

		return {
			leftMenuItems: leftMenuItems,
			rightMenuItems: rightMenuItems
		}
	},
	dashboard: function() {
		return <div className="container-fluid main"><Dashboard router={this.props.router} /></div>;
	},
	doctorSearch: function() {
		return <div className="container-fluid main"><DoctorSearch router={this.props.router} /></div>;
	},
	doctor: function() {
		return <Doctor router={this.props.router} doctorID={this.props.router.params.doctorID} page={this.props.router.params.page} />
	},
	guides: function() {
		return <Guides router={this.props.router} page={this.props.router.params.page} guideID={this.props.router.params.guideID} />
	},
	analytics: function() {
		return <Analytics router={this.props.router} page={this.props.router.params.page} reportID={this.props.router.params.reportID} />
	},
	accountsList: function() {
		return <AccountList router={this.props.router} accountID={this.props.router.params.accountID} />
	},
	account: function() {
		return <Account router={this.props.router} accountID={this.props.router.params.accountID} page={this.props.router.params.page} />
	},
	componentWillMount : function() {
		this.callback = (function() {
			this.forceUpdate();
		}).bind(this);
		this.props.router.on("route", this.callback);
	},
	componentWillUnmount : function() {
		this.props.router.off("route", this.callback);
	},
	render: function() {
		var page = this[this.props.router.current];
		return (
			<div>
				<TopNav router={this.props.router} leftItems={this.props.leftMenuItems} rightItems={this.props.rightMenuItems} activeItem={this.props.router.current} name="Admin" />
				{page ? page() : "Page Not Found"}
			</div>
		);
	}
});

// Pages

Dashboard = React.createClass({displayName: "Dashboard",
	mixins: [RouterNavigateMixin],
	getInitialState: function() {
		return {onboardURL: ""}
	},
	componentWillMount: function() {
		document.title = "Dashboard | Spruce Admin";
		this.onRefreshOnboardURL();
	},
	onRefreshOnboardURL: function() {
		AdminAPI.doctorOnboarding(function(success, res, jqXHR) {
			if (this.isMounted()) {
				if (!success) {
					console.error(jqXHR);
					this.setState({onboardURL: "FAILED"})
					return;
				}
				this.setState({onboardURL: res});
			}
		}.bind(this));
	},
	render: function() {
		return (
			<div>
				<div className="row">
					<div className="col-md-6 form-group">
						<label className="control-label" htmlFor="onboardURL">
							Doctor Onboarding URL <a href="#" onClick={this.onRefreshOnboardURL}><span className="glyphicon glyphicon-refresh"></span></a>
						</label>
						<input readOnly value={this.state.onboardURL} className="form-control section-name" />
					</div>
				</div>
			</div>
		);
	}
});

var DoctorSearch = React.createClass({displayName: "DoctorSearch",
	mixins: [RouterNavigateMixin],
	getInitialState: function() {
		return {
			query: "",
			results: null
		};
	},
	componentWillMount: function() {
		document.title = "Search | Doctors | Spruce Admin";
		var q = getParameterByName("q");
		if (q != this.state.query) {
			this.setState({query: q});
			this.search(q);
		}

		// TODO: need to make sure the page that was navigated to is this one
		// this.navCallback = (function() {
		// 	var q = getParameterByName("q");
		// 	if (q != this.state.query) {
		// 		this.setState({query: q});
		// 		this.search(q);
		// 	}
		// }).bind(this);
		// this.props.router.on("route", this.navCallback);
	},
	componentWillUnmount : function() {
		// this.props.router.off("route", this.navCallback);
	},
	search: function(q) {
		this.props.router.navigate("/doctors?q=" + encodeURIComponent(q), {replace: true}); // TODO: replacing until back tracking works
		if (q == "") {
			this.setState({results: null})
		} else {
			AdminAPI.searchDoctors(q, function(success, res, jqXHR) {
				if (this.isMounted()) {
					if (!success) {
						console.error(jqXHR);
						alert("ERROR");
						return;
					}
					this.setState({results: res.results});
				}
			}.bind(this));
		}
	},
	onSearchSubmit: function(e) {
		e.preventDefault();
		this.search(this.state.query);
		return false;
	},
	onQueryChange: function(e) {
		this.setState({query: e.target.value});
	},
	render: function() {
		return (
			<div className="container doctor-search">
				<div className="row">
					<div className="col-md-3">&nbsp;</div>
					<div className="col-md-6">
						<h2>Search doctors</h2>
						<form onSubmit={this.onSearchSubmit}>
							<div className="form-group">
								<input required autofocus type="text" className="form-control" name="q" value={this.state.query} onChange={this.onQueryChange} />
							</div>
							<button type="submit" className="btn btn-primary btn-lg center-block">Search</button>
						</form>
					</div>
					<div className="col-md-3">&nbsp;</div>
				</div>

				{this.state.results ? DoctorSearchResults({
					router: this.props.router,
					results: this.state.results}) : ""}
			</div>
		);
	}
});

var DoctorSearchResults = React.createClass({displayName: "DoctorSearchResult",
	mixins: [RouterNavigateMixin],
	render: function() {
		if (this.props.results.length == 0) {
			return (<div className="no-results">No Results</div>);
		}

		var results = this.props.results.map(function (res) {
			return (
				<div className="row" key={res.doctor_id}>
					<div className="col-md-3">&nbsp;</div>
					<div className="col-md-6">
						<DoctorSearchResult result={res} router={this.props.router} />
					</div>
					<div className="col-md-3">&nbsp;</div>
				</div>
			);
		}.bind(this))

		return (
			<div className="search-results">{results}</div>
		);
	}
});

var DoctorSearchResult = React.createClass({displayName: "DoctorSearchResult",
	mixins: [RouterNavigateMixin],
	render: function() {
		return (
			<a href={"doctors/"+this.props.result.doctor_id+"/info"} onClick={this.onNavigate}>{"Dr. "+this.props.result.first_name+" "+this.props.result.last_name+" ("+this.props.result.email+")"}</a>
		);
	}
});

var Doctor = React.createClass({displayName: "Doctor",
	menuItems: [[
		{
			id: "info",
			url: "info",
			name: "Info"
		},
		{
			id: "licenses",
			url: "licenses",
			name: "Licenses"
		},
		{
			id: "profile",
			url: "profile",
			name: "Profile"
		},
		{
			id: "savedmessage",
			url: "savedmessage",
			name: "Saved Message"
		}
	]],
	getInitialState: function() {
		return {
			doctor: null
		};
	},
	componentWillMount: function() {
		AdminAPI.doctor(this.props.doctorID, function(success, data) {
			if (this.isMounted()) {
				if (!success) {
					// TODO
					alert("Failed to fetch doctor");
					return;
				}
				document.title = data.doctor.short_display_name + " | Doctors | Spruce Admin";
				this.setState({doctor: data.doctor});
			}
		}.bind(this));
	},
	info: function() {
		return <DoctorInfoPage router={this.props.router} doctor={this.state.doctor} />;
	},
	licenses: function() {
		return <DoctorLicensesPage router={this.props.router} doctor={this.state.doctor} />;
	},
	profile: function() {
		return <DoctorProfilePage router={this.props.router} doctor={this.state.doctor} />;
	},
	savedmessage: function() {
		return <DoctorSavedMessagePage router={this.props.router} doctor={this.state.doctor} />;
	},
	render: function() {
		if (this.state.doctor == null) {
			// TODO
			return <div>LOADING</div>;
		}
		return (
			<div>
				<LeftNav router={this.props.router} items={this.menuItems} currentPage={this.props.page}>
					{this[this.props.page]()}
				</LeftNav>
			</div>
		);
	}
});

var DoctorInfoPage = React.createClass({displayName: "DoctorInfoPage",
	mixins: [RouterNavigateMixin],
	getInitialState: function() {
		return {
			updateAvatar: "",
			attributes: {},
			thumbnailURL: {
				"small": AdminAPI.doctorThumbnailURL(this.props.doctor.id, "small"),
				"large": AdminAPI.doctorThumbnailURL(this.props.doctor.id, "large")
			}
		};
	},
	componentWillMount: function() {
		document.title = this.props.doctor.short_display_name + " | Doctors | Spruce Admin";
		AdminAPI.doctorAttributes(this.props.doctor.id, function(success, data) {
			if (this.isMounted()) {
				if (!success) {
					// TODO
					alert("Failed to fetch doctor attributes");
					return;
				}
				this.setState({attributes: data});
			}
		}.bind(this));
	},
	onUpdate: function() {
		// Change the thumbnail URLs to force them to reload
		var v = Math.floor((Math.random() * 100000) + 1);
		this.setState({
			thumbnailURL: {
				"small": AdminAPI.doctorThumbnailURL(this.props.doctor.id, "small")+"?v="+v,
				"large": AdminAPI.doctorThumbnailURL(this.props.doctor.id, "large")+"?v="+v
			}
		});
	},
	render: function() {
		var createRow = function(attr) {
			var val = attr.value;
			if (attr.name == "DriversLicenseFile" || attr.name == "CVFile" || attr.name == "ClaimsHistoryFile") {
				val = <a href={"/admin/doctors/" + this.props.doctor.id + "/dl/" + attr.name}>Download</a>;
			}
			return (
				<tr key={attr.name}>
					<td><strong>{attr.name}</strong></td>
					<td>{val}</td>
				</tr>
			);
		}.bind(this);
		var attrList = [];
		for (var name in this.state.attributes) {
			attrList.push({name: name, value: this.state.attributes[name]});
		}
		attrList.sort(function(a, b){ return a.name > b.name; });
		return (
			<div>
				<DoctorUpdateThumbnailModal onUpdate={this.onUpdate} doctor={this.props.doctor} size="small" />
				<DoctorUpdateThumbnailModal onUpdate={this.onUpdate} doctor={this.props.doctor} size="large" />
				<h2>{this.props.doctor.long_display_name}</h2>
				<h3>Thumbnails</h3>
				<div className="row text-center">
					<div className="col-sm-6">
						<img src={this.state.thumbnailURL["small"]} className="doctor-thumbnail" />
						<br />
						Small
						<br />
						<button className="btn btn-default" data-toggle="modal" data-target="#avatarUpdateModal-small">
						Update
						</button>
					</div>
					<div className="col-sm-6">
						<img src={this.state.thumbnailURL["large"]} className="doctor-thumbnail" />
						<br />
						Large
						<br />
						<button className="btn btn-default" data-toggle="modal" data-target="#avatarUpdateModal-large">
						Update
						</button>
					</div>
				</div>
				<h3>General Info</h3>
				<table className="table">
					<tbody>
						<tr>
							<td><strong>NPI</strong></td>
							<td>{this.props.doctor.npi}</td>
						</tr>
						<tr>
							<td><strong>DEA</strong></td>
							<td>{this.props.doctor.dea}</td>
						</tr>
						{attrList.map(createRow)}
					</tbody>
				</table>
			</div>
		);
	}
});

var DoctorUpdateThumbnailModal = React.createClass({displayName: "DoctorUpdateThumbnailModal",
	onSubmit: function(e) {
		e.preventDefault();
		var formData = new FormData(e.target);
		AdminAPI.updateDoctorThumbnail(this.props.doctor.id, this.props.size, formData, function(success, data, jqXHR) {
			if (!success) {
				// TODO
				console.log(jqXHR);
				alert("Failed to upload thumbnail");
				return;
			}
			$("#avatarUpdateModal-"+this.props.size).modal('hide');
			this.props.onUpdate();
		}.bind(this));
		return false;
	},
	render: function() {
		return (
			<div className="modal fade" id={"avatarUpdateModal-"+this.props.size} role="dialog" tabIndex="-1">
				<div className="modal-dialog">
					<div className="modal-content">
						<form role="form" onSubmit={this.onSubmit}>
							<div className="modal-header">
								<button type="button" className="close" data-dismiss="modal"><span aria-hidden="true">&times;</span><span className="sr-only">Close</span></button>
								<h4 className="modal-title" id={"avatarUpdateModalTitle-"+this.props.size}>Update {this.props.size} Avatar</h4>
							</div>
							<div className="modal-body">
								<input required type="file" name="thumbnail" />
							</div>
							<div className="modal-footer">
								<button type="button" className="btn btn-default" data-dismiss="modal">Close</button>
								<button type="submit" className="btn btn-primary">Save</button>
							</div>
						</form>
					</div>
				</div>
			</div>
		);
	}
});

var DoctorLicensesPage = React.createClass({displayName: "DoctorLicensesPage",
	mixins: [RouterNavigateMixin],
	getInitialState: function() {
		return {licenses: []};
	},
	componentWillMount: function() {
		AdminAPI.medicalLicenses(this.props.doctor.id, function(success, licenses) {
			if (this.isMounted()) {
				if (success) {
					this.setState({licenses: licenses || []});
				} else {
					alert("Failed to get licenses");
				}
			}
		}.bind(this));
	},
	render: function() {
		function createRow(lic) {
			return (
				<tr key={lic.state}>
					<td>{lic.state}</td>
					<td>{lic.number}</td>
					<td>{lic.status}</td>
					<td>{lic.expiration}</td>
				</tr>
			);
		}
		return (
			<div>
				<h2>{this.props.doctor.long_display_name}</h2>
				<h3>Medical Licenses</h3>
				<table className="table">
					<thead>
						<tr>
							<th>State</th>
							<th>Number</th>
							<th>Status</th>
							<th>Expiration</th>
						</tr>
					</thead>
					<tbody>
						{this.state.licenses.map(createRow)}
					</tbody>
				</table>
			</div>
		);
	}
});

var doctorProfileFields = [
	{id: "short_title", label: "Short title", type: "text"},
	{id: "long_title", label: "Long title", type: "text"},
	{id: "short_display_name", label: "Short display name", type: "text"},
	{id: "long_display_name", label: "Long display name", type: "text"},
	{id: "full_name", label: "Full professional name", type: "text"},
	{id: "why_spruce", label: "Why Spruce?", type: "textarea"},
	{id: "qualifications", label: "Qualifications", type: "textarea"},
	{id: "medical_school", label: "Education :: Medical school", type: "textarea"},
	{id: "graduate_school", label: "Education :: Graduate school", type: "textarea"},
	{id: "undergraduate_school", label: "Education :: Undergraduate school", type: "textarea"},
	{id: "residency", label: "Education :: Residency", type: "textarea"},
	{id: "fellowship", label: "Education :: Fellowship", type: "textarea"},
	{id: "experience", label: "Experience", type: "textarea"}
];

var EditDoctorProfile = React.createClass({displayName: "EditDoctorProfile",
	mixins: [RouterNavigateMixin],
	getInitialState: function() {
		return {
			profile: {},
			error: "",
			modified: false,
			busy: false
		};
	},
	componentWillMount: function() {
		this.setState({profile: jQuery.extend({}, this.props.profile)});
	},
	componentWillUnmount: function() {
		// TODO: warn if modified before navigating away
	},
	onChange: function(e) {
		var profile = this.state.profile;
		profile[e.target.name] = e.target.value;
		this.setState({modified: true, profile: profile});
	},
	onCancel: function() {
		if (!this.state.busy) {
			this.props.onDone();
		}
		return false;
	},
	onSave: function() {
		if (this.state.busy) {
			return false;
		}
		if (!this.state.modified) {
			this.props.onDone();
			return false;
		}
		this.setState({busy: true});
		AdminAPI.updateCareProviderProfile(this.props.doctor.id, this.state.profile, function(success) {
			if (this.isMounted()) {
				this.setState({busy: false});
				if (success) {
					this.props.onDone();
				} else {
					this.setState({error: "Save failed"});
				}
			}
		}.bind(this));
		return false;
	},
	render: function() {
		var fields = [];
		for(var i = 0; i < doctorProfileFields.length; i++) {
			var f = doctorProfileFields[i];
			if (f.type == "textarea") {
				fields.push(TextArea({key: f.id, name: f.id, label: f.label, value: this.state.profile[f.id], onChange: this.onChange}));
			} else {
				fields.push(FormInput({key: f.id, name: f.id, type: f.type, label: f.label, value: this.state.profile[f.id], onChange: this.onChange}));
			}
		};
		var alert = null;
		if (this.state.error != "") {
			alert = Alert({"type": "danger"}, this.state.error);
		}
		var spinner = null;
		if (this.state.busy) {
			spinner = <img src={staticURL("/img/loading.gif")} />;
		}
		return (
			<div>
				<h2>{this.state.profile.long_display_name}</h2>
				<h3>Profile</h3>
				{fields}
				<div className="text-right">
					{alert}
					{spinner}
					<button className="btn btn-default" onClick={this.onCancel}>Cancel</button>
					&nbsp;
					<button className="btn btn-primary" onClick={this.onSave}>Save</button>
				</div>
			</div>
		);
	}
});

var DoctorProfilePage = React.createClass({displayName: "DoctorProfilePage",
	mixins: [RouterNavigateMixin],
	getInitialState: function() {
		return {
			busy: false,
			editing: false,
			profile: {}
		};
	},
	componentWillMount: function() {
		this.fetchProfile();
	},
	fetchProfile: function() {
		this.setState({busy: true});
		AdminAPI.careProviderProfile(this.props.doctor.id, function(success, profile) {
			if (success) {
				if (this.isMounted()) {
					this.setState({profile: profile, busy: false});
				}
			} else {
				// TODO
				alert("Failed to get profile")
			}
		}.bind(this));
	},
	edit: function() {
		this.setState({editing: true})
	},
	doneEditing: function() {
		this.setState({editing: false});
		this.fetchProfile();
	},
	render: function() {
		if (this.state.editing) {
			return EditDoctorProfile({
				doctor: this.props.doctor,
				profile: this.state.profile,
				onDone: this.doneEditing
			})
		}

		var fields = [];
		for(var i = 0; i < doctorProfileFields.length; i++) {
			var f = doctorProfileFields[i];
			fields.push(<br key={"br-"+f.id} />);
			if (f.type == "textarea") {
				fields.push(
					<div key={f.id} className="">
						<strong>{f.label}</strong><br />
						<pre>{this.state.profile[f.id]}</pre>
					</div>
				);
			} else {
				fields.push(
					<div key={f.id} className="row">
						<div className="col-md-3"><strong>{f.label}</strong></div>
						<div className="col-md-9">{this.state.profile[f.id]}</div>
					</div>
				);
			}
		};
		return (
			<div>
				<h2>{this.state.profile.long_display_name}</h2>
				<div className="pull-right">
					<button className="btn btn-default" onClick={this.edit}>Edit</button>
				</div>
				<h3>Profile</h3>
				{fields}
			</div>
		);
	}
});

var EditDoctorSavedMessage = React.createClass({displayName: "EditDoctorSavedMessage",
	mixins: [RouterNavigateMixin],
	getInitialState: function() {
		return {
			message: "",
			error: "",
			modified: false,
			busy: false
		};
	},
	componentWillMount: function() {
		this.setState({message: this.props.message});
	},
	componentWillUnmount: function() {
		// TODO: warn if modified before navigating away
	},
	onChange: function(e) {
		this.setState({modified: true, message: e.target.value});
	},
	onCancel: function() {
		if (!this.state.busy) {
			this.props.onDone();
		}
		return false;
	},
	onSave: function() {
		if (this.state.busy) {
			return false;
		}
		if (!this.state.modified) {
			this.props.onDone();
			return false;
		}
		this.setState({busy: true});
		AdminAPI.updateDoctorSavedMessage(this.props.doctor.id, this.state.message, function(success) {
			if (this.isMounted()) {
				this.setState({busy: false});
				if (success) {
					this.props.onDone();
				} else {
					this.setState({error: "Save failed"});
				}
			}
		}.bind(this));
		return false;
	},
	render: function() {
		var alert = null;
		if (this.state.error != "") {
			alert = Alert({"type": "danger"}, this.state.error);
		}
		var spinner = null;
		if (this.state.busy) {
			spinner = <img src={staticURL("/img/loading.gif")} />;
		}
		return (
			<div>
				<h2>{this.props.doctor.long_display_name}</h2>
				<h3>Saved Message</h3>
				<TextArea name="saved-message" label="" value={this.state.message} rows="15" onChange={this.onChange} />
				<div className="text-right">
					{alert}
					{spinner}
					<button className="btn btn-default" onClick={this.onCancel}>Cancel</button>
					&nbsp;
					<button className="btn btn-primary" onClick={this.onSave}>Save</button>
				</div>
			</div>
		);
	}
});

var DoctorSavedMessagePage = React.createClass({displayName: "DoctorSavedMessagePage",
	mixins: [RouterNavigateMixin],
	getInitialState: function() {
		return {
			editing: false,
			message: ""
		};
	},
	componentWillMount: function() {
		this.fetchMessage();
	},
	fetchMessage: function() {
		this.setState({busy: true});
		AdminAPI.doctorSavedMessage(this.props.doctor.id, function(success, data) {
			if (success) {
				if (this.isMounted()) {
					this.setState({message: data.message})
				}
			} else {
				// TODO
				alert("Failed to get saved message")
			}
		}.bind(this));
	},
	edit: function() {
		this.setState({editing: true})
	},
	doneEditing: function() {
		this.setState({editing: false});
		this.fetchMessage();
	},
	render: function() {
		if (this.state.editing) {
			return EditDoctorSavedMessage({
				doctor: this.props.doctor,
				message: this.state.message,
				onDone: this.doneEditing
			})
		}

		return (
			<div>
				<h2>{this.props.doctor.long_display_name}</h2>
				<div className="pull-right">
					<button className="btn btn-default" onClick={this.edit}>Edit</button>
				</div>
				<h3>Saved Message</h3>
				<pre>{this.state.message}</pre>
			</div>
		);
	}
});

var Guides = React.createClass({displayName: "Guides",
	menuItems: [[
		{
			id: "resources",
			url: "/admin/guides/resources",
			name: "Resource Guides"
		},
		{
			id: "rx",
			url: "/admin/guides/rx",
			name: "RX Guides"
		}
	]],
	getDefaultProps: function() {
		return {
			guideID: null
		}
	},
	resources: function() {
		if (this.props.guideID != null) {
			return <ResourceGuide router={this.props.router} guideID={this.props.guideID} />;
		}
		return <ResourceGuideList router={this.props.router} />;
	},
	rx: function() {
		if (this.props.guideID != null) {
			return <RXGuide router={this.props.router} ndc={this.props.guideID} />;
		}
		return <RXGuideList router={this.props.router} />;
	},
	render: function() {
		return (
			<div>
				<LeftNav router={this.props.router} items={this.menuItems} currentPage={this.props.page}>
					{this[this.props.page]()}
				</LeftNav>
			</div>
		);
	}
});

var ResourceGuide = React.createClass({displayName: "ResourceGuide",
	mixins: [RouterNavigateMixin],
	getInitialState: function() {
		return {
			guide: {},
			sections: [],
			error: ""
		};
	},
	componentWillMount: function() {
		AdminAPI.resourceGuide(this.props.guideID, function(success, data) {
			if (this.isMounted()) {
				if (success) {
					document.title = data.title + " | Resources | Guides | Spruce Admin";
					data.layout_json = JSON.stringify(data.layout, null, 4);
					this.setState({guide: data});
				} else {
					alert("Failed to get resource guide");
				}
			}
		}.bind(this));
		AdminAPI.resourceGuidesList(false, true, function(success, data) {
			if (this.isMounted()) {
				if (success) {
					this.setState({sections: data.sections});
				} else {
					alert("Failed to get sections");
				}
			}
		}.bind(this));
	},
	onChange: function(e) {
		var guide = this.state.guide;
		var val = e.target.value;
		// Make sure to maintain types
		if (typeof guide[e.target.name] == "number") {
			val = Number(val);
		}
		guide[e.target.name] = val;
		this.setState({error: "", guide: guide});
		return false;
	},
	onSubmit: function() {
		try {
			var guide = this.state.guide;
			var js = JSON.parse(guide.layout_json);
			guide.layout = js;
			this.setState({error: "", guide: guide});
		} catch (err) {
			this.setState({error: "Invalid layout: " + err.message});
			return false;
		};

		AdminAPI.updateResourceGuide(this.props.guideID, this.state.guide, function(success, data, jqXHR) {
			if (this.isMounted()) {
				if (!success) {
					console.error(jqXHR);
					alert("Failed to save resource guide");
				}
			}
		}.bind(this));
		return false;
	},
	render: function() {
		var sectionOptions = this.state.sections.map(function(s) {
			return {value: s.id, name: s.title};
		})
		return (
			<div className="resource-guide-edit">
				<h2><img src={this.state.guide.photo_url} width="32" height="32" /> {this.state.guide.title}</h2>

				<form role="form" onSubmit={this.onSubmit} method="PUT">
					<div className="row">
						<div className="col-md-2">
							<FormSelect name="section_id" label="Section" value={this.state.guide.section_id} opts={sectionOptions} onChange={this.onChange} />
						</div>
						<div className="col-md-2">
							<FormInput name="ordinal" type="number" required label="Ordinal" value={this.state.guide.ordinal} onChange={this.onChange} />
						</div>
						<div className="col-md-8">
							<FormInput name="photo_url" type="url" required label="Photo URL" value={this.state.guide.photo_url} onChange={this.onChange} />
						</div>
					</div>
					<div>
						<FormInput name="title" type="text" required label="Title" value={this.state.guide.title} onChange={this.onChange} />
					</div>
					<div>
						<TextArea name="layout_json" required label="Layout" value={this.state.guide.layout_json} rows="30" onChange={this.onChange} />
					</div>
					<div className="text-right">
						{this.state.error ? <Alert type="danger">{this.state.error}</Alert> : null}
						<button type="submit" className="btn btn-primary">Save</button>
					</div>
				</form>
			</div>
		);
	}
});

var ResourceGuideList = React.createClass({displayName: "ResourceGuideList",
	mixins: [RouterNavigateMixin],
	getInitialState: function() {
		return {sections: []};
	},
	componentWillMount: function() {
		document.title = "Resources | Guides | Spruce Admin";
		this.updateList();
	},
	updateList: function() {
		AdminAPI.resourceGuidesList(false, false, function(success, data) {
			if (this.isMounted()) {
				if (success) {
					var sections = data.sections;
					for(var i = 0; i < sections.length; i++) {
						var s = sections[i];
						s.guides = data.guides[s.id];
					}
					this.setState({sections: sections});
				} else {
					alert("Failed to get resource guides");
				}
			}
		}.bind(this));
	},
	onImport: function(e) {
		e.preventDefault();
		var formData = new FormData(e.target);
		AdminAPI.resourceGuidesImport(formData, function(success, data, jqXHR) {
			if (!success) {
				// TODO
				console.log(jqXHR);
				alert("Failed to import resource guides");
				return;
			}
			this.updateList();
		}.bind(this));
		return false;
	},
	onExport: function(e) {
		e.preventDefault();
		AdminAPI.resourceGuidesExport(function(success, data) {
			if (this.isMounted()) {
				if (success) {
					var pom = document.createElement('a');
					pom.setAttribute('href', 'data:application/json;charset=utf-8,' + encodeURIComponent(data));
					pom.setAttribute('download', "resource_guides.json");
					pom.click();
				} else {
					alert("Failed to get resource guides");
				}
			}
		}.bind(this));
		return false;
	},
	render: function() {
		var t = this;
		var createSection = function(section) {
			return (
				<div key={section.id}>
					<Section router={this.props.router} section={section} />
				</div>
			);
		}.bind(this);
		return (
			<div>
				<ModalForm id="import-resource-guides-modal" title="Import Resource Guides" cancelButtonTitle="Cancel" submitButtonTitle="Import" onSubmit={this.onImport}>
					<input required type="file" name="json" />
				</ModalForm>
				<div className="pull-right">
					<button className="btn btn-default" data-toggle="modal" data-target="#import-resource-guides-modal">Import</button>
					&nbsp;
					<button className="btn btn-default" onClick={this.onExport}>Export</button>
				</div>
				<div>{this.state.sections.map(createSection)}</div>
			</div>
		);
	}
});

var Section = React.createClass({displayName: "Section",
	mixins: [RouterNavigateMixin],
	getInitialState: function() {
		return {editing: false};
	},
	render: function() {
		var createGuideItem = function(guide) {
			guide.key = guide.id;
			guide.router = this.props.router;
			return GuideListItem(guide);
		}.bind(this);
		var title;
		if (this.state.editing) {
			title = <input type="text" className="form-control section-name" value={this.props.section.title} />;
		} else {
			title = <h4>{this.props.section.title}</h4>;
		}
		return (
			<div className="section">
				{title}
				{this.props.section.guides.map(createGuideItem)}
			</div>
		);
	}
});

var GuideListItem = React.createClass({displayName: "GuideListItem",
	mixins: [RouterNavigateMixin],
	render: function() {
		return (
			<div key={this.props.id} className="item">
				<img src={this.props.photo_url} width="32" height="32" />
				&nbsp;<a href={"resources/"+this.props.id} onClick={this.onNavigate}>{this.props.title}</a>
			</div>
		);
	}
});

var RXGuide = React.createClass({displayName: "RXGuide",
	mixins: [RouterNavigateMixin],
	getInitialState: function() {
		return {"guide": []}
	},
	componentWillMount: function() {
		AdminAPI.rxGuide(this.props.ndc, true, function(success, data) {
			if (this.isMounted()) {
				if (success) {
					document.title = this.props.ndc + " | RX | Guides | Spruce Admin";
					this.setState({guide: data.guide, html: data.html});
				} else {
					alert("Failed to get rx guide");
				}
			}
		}.bind(this));
	},
	render: function() {
		return (
			<div className="rxguide">
				<div dangerouslySetInnerHTML={{__html: this.state.html}}></div>
			</div>
		);
	}
});

var RXGuideList = React.createClass({displayName: "RXGuideList",
	mixins: [RouterNavigateMixin],
	getInitialState: function() {
		return {"guides": []}
	},
	componentWillMount: function() {
		document.title = "RX | Guides | Spruce Admin";
		this.updateList();
	},
	updateList: function() {
		AdminAPI.rxGuidesList(function(success, data) {
			if (this.isMounted()) {
				if (success) {
					this.setState({guides: data});
				} else {
					alert("Failed to get rx guides");
				}
			}
		}.bind(this));
	},
	onImport: function(e) {
		e.preventDefault();
		var formData = new FormData(e.target);
		AdminAPI.rxGuidesImport(formData, function(success, data, jqXHR) {
			if (!success) {
				// TODO
				console.log(jqXHR);
				alert("Failed to import rx guides");
				return;
			}
			this.updateList();
		}.bind(this));
		return false;
	},
	render: function() {
		return (
			<div className="rx-guide-list">
				<ModalForm id="import-rx-guides-modal" title="Import RX Guides" cancelButtonTitle="Cancel" submitButtonTitle="Import" onSubmit={this.onImport}>
					<input required type="file" name="csv" />
				</ModalForm>
				<div className="pull-right">
					<button className="btn btn-default" data-toggle="modal" data-target="#import-rx-guides-modal">Import</button>
				</div>

				<h2>RX Guides</h2>
				{this.state.guides.map(function(guide) {
					return <div key={guide.NDC} className="rx-guide">
						<a href={"/admin/guides/rx/" + guide.NDC} onClick={this.onNavigate}>{guide.Name + " (" + guide.NDC + ")"}</a>
					</div>
				}.bind(this))}
			</div>
		);
	}
});

var Analytics = React.createClass({displayName: "Analytics",
	mixins: [RouterNavigateMixin],
	getInitialState: function() {
		return {
			reports: [],
			menuItems: this.defaultMenuItems(),
		};
	},
	defaultMenuItems: function() {
		var menuItems = [];
		if (Perms.has(Perms.AnalyticsReportsEdit)) {
			menuItems.push([
				{
					id: "query",
					url: "/admin/analytics/query",
					name: "Query"
				}
			]);
		}
		return menuItems;
	},
	componentWillMount: function() {
		// TODO: use ace editor for syntax highlighting
		// var script = document.createElement("script");
		// script.setAttribute("src", "https://cdnjs.cloudflare.com/ajax/libs/ace/1.1.3/ace.js")
		// document.head.appendChild(script);

		document.title = "Analytics | Spruce Admin";

		this.loadReports();
	},
	loadReports: function() {
		AdminAPI.listAnalyticsReports(function(success, data, jqXHR) {
			if (this.isMounted()) {
				if (!success) {
					console.error(jqXHR);
					alert("Failed to get reports list");
					return;
				}
				data = data || [];
				var repMenu = [];
				for(var i = 0; i < data.length; i++) {
					var rep = data[i];
					repMenu.push({
						id: "report-" + rep.id,
						url: "/admin/analytics/reports/" + rep.id,
						name: rep.name
					});
				}
				var menuItems = this.defaultMenuItems();
				menuItems.push(repMenu);
				this.setState({
					reports: data,
					menuItems: menuItems
				});

				if (this.props.page == "query" && !Perms.has(Perms.AnalyticsReportsEdit)) {
					this.navigate("/analytics/reports/" + data[0].id);
				}
			}
		}.bind(this));
	},
	onSaveReport: function(report) {
		this.loadReports();
	},
	query: function() {
		if (!Perms.has(Perms.AnalyticsReportsEdit)) {
			return <div></div>;
		}
		return <AnalyticsQuery router={this.props.router} />;
	},
	reports: function() {
		return <AnalyticsReport router={this.props.router} reportID={this.props.reportID} onSave={this.onSaveReport} />;
	},
	render: function() {
		// TODO: this is janky
		var currentPage = this.props.page;
		if (currentPage == "reports") {
			currentPage = "report-" + this.props.reportID;
		}
		return (
			<div>
				<LeftNav router={this.props.router} items={this.state.menuItems} currentPage={currentPage}>
					{this[this.props.page]()}
				</LeftNav>
			</div>
		);
	}
});

function DownloadAnalyticsCSV(results, name) {
	// Generate CSV of the results
	var csv = results.cols.join(",");
	for(var i = 0; i < results.rows.length; i++) {
		var row = results.rows[i];
		csv += "\n" + row.map(function(v) {
			if (typeof v == "number") {
				return v;
			} else if (typeof v == "string") {
				return '"' + v.replace(/"/g, '""') + '"';
			} else {
				return '"' + v.toString().replace(/"/g, '""') + '"';
			}
		}).join(",");
	}

	name = name || "analytics";

	var pom = document.createElement('a');
	pom.setAttribute('href', 'data:text/csv;charset=utf-8,' + encodeURIComponent(csv));
	pom.setAttribute('download', name + ".csv");
	pom.click();
}

var AnalyticsQuery = React.createClass({displayName: "AnalyticsQuery",
	mixins: [RouterNavigateMixin],
	getInitialState: function() {
		return {
			query: "",
			presentation: "",
			error: "",
			running: false,
			results: null
		};
	},
	query: function(q) {
		if (q == "") {
			this.setState({error: "", results: null})
		} else {
			this.setState({running: true, error: ""});
			AdminAPI.analyticsQuery(q, function(success, res, jqXHR) {
				if (this.isMounted()) {
					this.setState({running: false});
					if (!success) {
						console.error(jqXHR);
						alert("ERROR");
						return;
					}
					if (res.error) {
						this.setState({error: res.error, results: null})
					} else {
						this.setState({
							error: "",
							results: {
								cols: res.cols,
								rows: res.rows
							}
						});
					}
				}
			}.bind(this));
		}
	},
	onSubmit: function(e) {
		e.preventDefault();
		this.query(this.state.query);
		return false;
	},
	onQueryChange: function(e) {
		this.setState({query: e.target.value});
	},
	onDownload: function(e) {
		e.preventDefault();
		DownloadAnalyticsCSV(this.state.results);
		return false;
	},
	onSave: function(e) {
		e.preventDefault();
		AdminAPI.createAnalyticsReport("New Report", this.state.query, "", function(success, reportID) {
			if (this.isMounted()) {
				if (!success) {
					this.setState({error: "Failed to save report"});
					return;
				}
				this.navigate("/analytics/reports/" + reportID);
			}
		}.bind(this));
		return false;
	},
	render: function() {
		return (
			<div className="analytics">
				<div className="form">
					<div className="text-center">
						<h2>Analytics</h2>
					</div>
					<form onSubmit={this.onSubmit}>
						<TextArea tabs="true" label="Query" name="q" value={this.state.query} onChange={this.onQueryChange} rows="10" />
						<div className="text-center">
							<button className="btn btn-default" onClick={this.onSave}>Save</button>
							&nbsp;<button disabled={this.state.results ? "" : "disabled"} className="btn btn-default" onClick={this.onDownload}>Download</button>
							&nbsp;<button type="submit" className="btn btn-primary">Query</button>
						</div>
					</form>
				</div>

				{this.state.error ? <Alert type="danger">{this.state.error}</Alert> : null}

				{this.state.running ? <Alert type="info">Querying... please wait</Alert> : null}

				{this.state.results ? AnalyticsResults({
					router: this.props.router,
					results: this.state.results
				}) : null}
			</div>
		);
	}
});

var AnalyticsQueryCache = {};

var AnalyticsReport = React.createClass({displayName: "AnalyticsReport",
	mixins: [RouterNavigateMixin],
	getInitialState: function() {
		return {
			id: null,
			name: "",
			query: "",
			presentation: "",
			error: "",
			running: false,
			results: null,
			version: 1,
			editing: false
		};
	},
	componentWillMount: function() {
		this.loadReport(this.props.reportID);
	},
	componentWillReceiveProps: function(nextProps) {
		if (this.props.reportID != nextProps.reportID) {
			this.loadReport(nextProps.reportID);
		}
	},
	componentWillUpdate: function(nextProps, nextState) {
		document.analyticsData = nextState.results;
	},
	loadReport: function(id) {
		AdminAPI.analyticsReport(id, function(success, report, jqXHR) {
			if (this.isMounted()) {
				if (!success) {
					this.setState({error: "Failed to load report"})
					console.error(jqXHR);
					return
				}
				document.title = report.name + " | Analytics | Spruce Admin";
				this.setState({
					id: report.id,
					name: report.name,
					query: report.query,
					presentation: report.presentation,
					error: "",
					results: AnalyticsQueryCache[report.id],
					editing: report.name == "New Report",
					version: this.state.version+1
				});
			}
		}.bind(this));
	},
	query: function(q) {
		if (q == "") {
			this.setState({error: "", results: null})
		} else {
			this.setState({running: true, error: ""});
			AdminAPI.analyticsQuery(q, function(success, res, jqXHR) {
				if (this.isMounted()) {
					this.updateResults(success, res, jqXHR)
				}
			}.bind(this));
		}
	},
	updateResults: function(success, res, jqXHR) {
		this.setState({running: false});
		if (!success) {
			console.error(jqXHR);
			alert("ERROR");
			return;
		}
		if (res.error) {
			this.setState({error: res.error, results: null})
		} else {
			results = {
				cols: res.cols,
				rows: res.rows
			}
			this.setState({
				error: "",
				results: results
			});
			AnalyticsQueryCache[this.state.id] = results;
			// TODO: push changes to presentation
			// var pres = this.refs.presentation;
			// if (pres != null) {
			// 	var onUpdate = pres.getDOMNode().onUpdate;
			// }
			// TODO: for now just force the iframe to reload
			this.setState({version: this.state.version+1});
		}
	},
	onSubmit: function(e) {
		e.preventDefault();
		this.query(this.state.query);
		return false;
	},
	onNameChange: function(e) {
		this.setState({name: e.target.value});
	},
	onQueryChange: function(e) {
		this.setState({query: e.target.value});
	},
	onPresentationChange: function(e) {
		this.setState({presentation: e.target.value});
	},
	onDownload: function(e) {
		e.preventDefault();
		DownloadAnalyticsCSV(this.state.results, this.state.name);
		return false;

	},
	onSave: function(e) {
		e.preventDefault();
		AdminAPI.updateAnalyticsReport(this.props.reportID, this.state.name, this.state.query, this.state.presentation, function(success, data) {
			if (this.isMounted()) {
				if (!success) {
					this.setState({error: "Failed to save report"});
					return;
				}
				if (this.props.onSave) {
					this.props.onSave({
						id: this.props.reportID,
						error: "",
						name: this.state.name,
						query: this.state.query,
						presentation: this.state.presentation,
					});
				}
				this.setState({version: this.state.version+1});
			}
		}.bind(this));
		return false;
	},
	onEdit: function(e) {
		e.preventDefault();
		this.setState({editing: true});
		return false;
	},
	onRun: function(e) {
		e.preventDefault();
		this.setState({running: true, error: ""});
		AdminAPI.runAnalyticsReport(this.props.reportID, function(success, data, jqXHR) {
			if (this.isMounted()) {
				this.updateResults(success, data, jqXHR);
			}
		}.bind(this));
		return false;
	},
	render: function() {
		// TODO: sandbox the iframe further by not allowing same-origin
		var form = null;
		if (this.state.editing) {
			form = (
				<div className="form">
					<form onSubmit={this.onSubmit}>
						<FormInput required type="text" label="Name" name="name" value={this.state.name} onChange={this.onNameChange} />
						<TextArea tabs="true" label="Query" name="query" value={this.state.query} onChange={this.onQueryChange} rows="10" />
						<TextArea tabs="true" label="Presentation" name="presentation" value={this.state.presentation} onChange={this.onPresentationChange} rows="15" />
						<div className="text-center">
							<button className="btn btn-default" onClick={this.onSave}>Save</button>
							&nbsp;<button disabled={this.state.results ? "" : "disabled"} className="btn btn-default" onClick={this.onDownload}>Download</button>
							&nbsp;<button type="submit" className="btn btn-primary">Query</button>
						</div>
					</form>
				</div>
			);
		}
		return (
			<div className="analytics">
				<div className="text-center">
					<h2>{this.state.name}</h2>
				</div>

				{this.state.editing ? form :
					<div className="form text-center">
						{Perms.has(Perms.AnalyticsReportsEdit) ? <button className="btn btn-default" onClick={this.onEdit}>Edit</button> : null}
						&nbsp;<button disabled={this.state.results ? "" : "disabled"} className="btn btn-default" onClick={this.onDownload}>Download</button>
						&nbsp;<button className="btn btn-primary" onClick={this.onRun}>Run</button>
					</div>}

				{this.state.error ? <Alert type="danger">{this.state.error}</Alert> : null}

				{this.state.running ? <Alert type="info">Querying... please wait</Alert> : null}

				{this.state.results && this.state.presentation ?
					<iframe sandbox="allow-scripts allow-same-origin" ref="presentation" src={"/admin/analytics/reports/"+this.props.reportID+"/presentation/iframe?v=" + this.state.version} border="0" width="100%" height="100%" />
					: null}

				{this.state.results ? AnalyticsResults({
					router: this.props.router,
					results: this.state.results
				}) : null}
			</div>
		);
	}
});

var AnalyticsResults = React.createClass({displayName: "AnalyticsResults",
	render: function() {
		analyticsData = this.props.results;
		return (
			<div className="analytics-results">
				<div className="text-right">
					{this.props.results.rows.length} rows
				</div>
				<table className="table">
					<thead>
						<tr>
						{this.props.results.cols.map(function(col) {
							return <th key={col}>{col}</th>;
						}.bind(this))}
						</tr>
					</thead>
					<tbody>
						{this.props.results.rows.map(function(row, indexRow) {
							return (
								<tr key={"analytics-query-row-"+indexRow}>
									{row.map(function(v, indexVal) {
										return <td key={"analytics-query-row-"+indexRow+"-"+indexVal}>{v}</td>;
									}.bind(this))}
								</tr>
							);
						}.bind(this))}
					</tbody>
				</table>
			</div>
		)
	}
});


////////////////////////////////////////// Admin Accounts /////////////////////////////////////////////


var AccountList = React.createClass({displayName: "AccountList",
	mixins: [RouterNavigateMixin],
	getInitialState: function() {
		return {
			query: "",
			results: null
		};
	},
	search: function(q) {
		if (q == "") {
			this.setState({results: null})
		} else {
			AdminAPI.searchAdmins(q, function(success, res, jqXHR) {
				if (this.isMounted()) {
					if (!success) {
						console.error(jqXHR);
						alert("ERROR");
						return;
					}
					this.setState({results: res.accounts || []});
				}
			}.bind(this));
		}
	},
	onSearchSubmit: function(e) {
		e.preventDefault();
		this.search(this.state.query);
		return false;
	},
	onQueryChange: function(e) {
		this.setState({query: e.target.value});
	},
	render: function() {
		return (
			<div className="container accounts-search">
				<div className="row">
					<div className="col-md-3">&nbsp;</div>
					<div className="col-md-6">
						<h2>Search admin accounts</h2>
						<form onSubmit={this.onSearchSubmit}>
							<div className="form-group">
								<input required autofocus type="email" className="form-control" name="q" value={this.state.query} onChange={this.onQueryChange} />
							</div>
							<button type="submit" className="btn btn-primary btn-lg center-block">Search</button>
						</form>
					</div>
					<div className="col-md-3">&nbsp;</div>
				</div>

				{this.state.results ? AccountSearchResults({
					router: this.props.router,
					results: this.state.results}) : ""}
			</div>
		);
	}
});


var AccountSearchResults = React.createClass({displayName: "AccountSearchResults",
	mixins: [RouterNavigateMixin],
	render: function() {
		if (this.props.results.length == 0) {
			return (<div className="no-results">No Results</div>);
		}

		var results = this.props.results.map(function (res) {
			return (
				<div className="row" key={res.id}>
					<div className="col-md-3">&nbsp;</div>
					<div className="col-md-6">
						<AccountSearchResult result={res} router={this.props.router} />
					</div>
					<div className="col-md-3">&nbsp;</div>
				</div>
			);
		}.bind(this))

		return (
			<div className="search-results">{results}</div>
		);
	}
});

var AccountSearchResult = React.createClass({displayName: "AccountSearchResult",
	mixins: [RouterNavigateMixin],
	render: function() {
		return (
			<a href={"accounts/"+this.props.result.id+"/permissions"} onClick={this.onNavigate}>{this.props.result.email}</a>
		);
	}
});

///

var Account = React.createClass({displayName: "Account",
	menuItems: [[
		{
			id: "permissions",
			url: "permissions",
			name: "Permissions"
		}
	]],
	getInitialState: function() {
		return {
			account: null
		};
	},
	componentWillMount: function() {
		AdminAPI.adminAccount(this.props.accountID, function(success, data) {
			if (this.isMounted()) {
				if (!success) {
					// TODO
					alert("Failed to fetch account");
					return;
				}
				this.setState({account: data.account});
			}
		}.bind(this));
	},
	permissions: function() {
		return <AccountPermissionsPage router={this.props.router} account={this.state.account} />;
	},
	render: function() {
		if (this.state.account == null) {
			// TODO
			return <div>LOADING</div>;
		}
		return (
			<div>
				<LeftNav router={this.props.router} items={this.menuItems} currentPage={this.props.page}>
					{this[this.props.page]()}
				</LeftNav>
			</div>
		);
	}
});

var AccountPermissionsPage = React.createClass({displayName: "AccountPermissionsPage",
	mixins: [RouterNavigateMixin],
	getInitialState: function() {
		return {
			groups: [],
			permissions: []
		};
	},
	componentWillMount: function() {
		this.loadGroups();
		this.loadPermissions();
	},
	loadGroups: function() {
		AdminAPI.adminGroups(this.props.account.id, function(success, data) {
			if (this.isMounted()) {
				if (!success) {
					// TODO
					alert("Failed to fetch account groups");
					return;
				}
				this.setState({groups: data.groups.sort(function(a, b) { return a.name > b.name; })});
			}
		}.bind(this));
	},
	loadPermissions: function() {
		AdminAPI.adminPermissions(this.props.account.id, function(success, data) {
			if (this.isMounted()) {
				if (!success) {
					// TODO
					alert("Failed to fetch account permissions");
					return;
				}
				this.setState({permissions: data.permissions.sort(function(a, b) { return a > b; })});
			}
		}.bind(this));
	},
	updateGroups: function(updates) {
		AdminAPI.updateAdminGroups(this.props.account.id, updates, function(success, data) {
			if (this.isMounted()) {
				if (!success) {
					// TODO
					alert("Failed to update permissions");
					return;
				}
				this.loadGroups();
				this.loadPermissions();
			}
		}.bind(this));
	},
	onRemoveGroup: function(group) {
		var updates = {};
		updates[group] = false;
		this.updateGroups(updates);
	},
	onAddGroup: function(group) {
		var updates = {};
		updates[group] = true;
		this.updateGroups(updates);
	},
	render: function() {
		return (
			<div>
				<h2>{this.props.account.email}</h2>
				<h3>Groups</h3>
				<AccountGroups allowEdit={Perms.has(Perms.AdminAccountsEdit)} onAdd={this.onAddGroup} onRemove={this.onRemoveGroup} groups={this.state.groups} />
				<h3>Permissions</h3>
				<AccountPermissions permissions={this.state.permissions} />
			</div>
		);
	}
});

var AccountGroups = React.createClass({displayName: "AccountGroups",
	getDefaultProps: function() {
		return {
			groups: [],
			allowEdit: false
		};
	},
	getInitialState: function() {
		return {
			adding: false,
			addingValue: null,
			availableGroups: []
		};
	},
	componentWillMount: function() {
		AdminAPI.availableGroups(true, function(success, data) {
			if (this.isMounted()) {
				if (!success) {
					// TODO
					alert("Failed to fetch available groups");
					return;
				}
				var groupOptions = data.groups.map(function(g) { return {value: g.id, name: g.name} });
				this.setState({
					availableGroups: data.groups,
					groupOptions: groupOptions,
					addingValue: groupOptions[0].value
				});
			}
		}.bind(this));
	},
	onAdd: function(e) {
		e.preventDefault();
		this.setState({adding: true});
		return false;
	},
	onChange: function(e) {
		this.setState({addingValue: e.target.value});
	},
	onCancel: function(e) {
		e.preventDefault();
		this.setState({adding: false});
		return false;
	},
	onSubmit: function(e) {
		e.preventDefault();
		this.props.onAdd(this.state.addingValue);
		this.setState({adding: false, addingValue: this.state.groupOptions[0].value});
		return false;
	},
	onRemove: function(group) {
		this.props.onRemove(group);
		return false;
	},
	render: function() {
		return (
			<div className="groups">
				{this.props.groups.map(function(group) {
					return (
						<div key={group.id}>
							{this.props.allowEdit ? <a href="#" onClick={this.onRemove.bind(this, group.id)}><span className="glyphicon glyphicon-remove" /></a> : null} {group.name}
						</div>
					);
				}.bind(this))}
				{this.state.adding ?
					<div>
						<form onSubmit={this.onSubmit}>
							<FormSelect onChange={this.onChange} value={this.addingValue} opts={this.state.groupOptions} />
							<button onClick={this.onCancel} className="btn btn-default">Cancel</button>
							&nbsp;<button type="submit" className="btn btn-default">Save</button>
						</form>
					</div> : null}
				{this.props.allowEdit && !this.state.adding ?
					<div><a href="#" onClick={this.onAdd}><span className="glyphicon glyphicon-plus" /></a></div> : null}
			</div>
		);
	}
});

var AccountPermissions = React.createClass({displayName: "AccountPermissions",
	getDefaultProps: function() {
		return {
			permissions: []
		};
	},
	render: function() {
		return (
			<div className="permissions">
				{this.props.permissions.map(function(perm) {
					return <div key={perm}>{perm}</div>;
				}.bind(this))}
			</div>
		);
	}
});

var ModalForm = React.createClass({displayName: "ModalForm",
	propTypes: {
		id: React.PropTypes.string.isRequired,
		title: React.PropTypes.string.isRequired,
		cancelButtonTitle: React.PropTypes.string.isRequired,
		submitButtonTitle: React.PropTypes.string.isRequired,
		onSubmit: React.PropTypes.func.isRequired
	},
	onSubmit: function(e) {
		e.preventDefault();
		this.props.onSubmit(e);
		$("#"+this.props.id).modal('hide');
		return false;
	},
	render: function() {
		return (
			<div className="modal fade" id={this.props.id} role="dialog" tabIndex="-1">
				<div className="modal-dialog">
					<div className="modal-content">
						<form role="form" onSubmit={this.onSubmit}>
							<div className="modal-header">
								<button type="button" className="close" data-dismiss="modal"><span aria-hidden="true">&times;</span><span className="sr-only">Close</span></button>
								<h4 className="modal-title">{this.props.title}</h4>
							</div>
							<div className="modal-body">
								{this.props.children}
							</div>
							<div className="modal-footer">
								<button type="button" className="btn btn-default" data-dismiss="modal">{this.props.cancelButtonTitle}</button>
								<button type="submit" className="btn btn-primary">{this.props.submitButtonTitle}</button>
							</div>
						</form>
					</div>
				</div>
			</div>
		);
	}
});

//////////////////// Form fields and utilities ///////////////////////

var FormSelect = React.createClass({displayName: "FormSelect",
	getDefaultProps: function() {
		return {opts: []};
	},
	render: function() {
		return (
			<div className="form-group">
				<label className="control-label" htmlFor={this.props.name}>{this.props.label}</label><br />
				<select name={this.props.name} className="form-control" value={this.props.value} onChange={this.props.onChange}>
					{this.props.opts.map(function(opt) {
						return <option key={"select-value-" + opt.value} value={opt.value}>{opt.name}</option>
					}.bind(this))}
				</select>
			</div>
		);
	}
});

var FormInput = React.createClass({displayName: "FormInput",
	getDefaultProps: function() {
		return {type: "text"}
	},
	render: function() {
		return (
			<div className="form-group">
				<label className="control-label" htmlFor={this.props.name}>{this.props.label}</label>
				<input type={this.props.type} className="form-control section-name" onKeyDown={this.props.onKeyDown}
					   name={this.props.name} value={this.props.value} onChange={this.props.onChange} />
			</div>
		);
	}
});

var TextArea = React.createClass({displayName: "TextArea",
	getDefaultProps: function() {
		return {
			rows: 5,
			tabs: false
		}
	},
	onKeyDown: function(e) {
		if (!this.props.tabs) {
			return;
		}
		var keyCode = e.keyCode || e.which;
		if (keyCode == 9) {
			e.preventDefault();
			var start = $(e.target).get(0).selectionStart;
			var end = $(e.target).get(0).selectionEnd;
			$(e.target).val($(e.target).val().substring(0, start) + "\t" + $(e.target).val().substring(end));
			$(e.target).get(0).selectionStart =
			$(e.target).get(0).selectionEnd = start + 1;
			return false;
		  }
	},
	render: function() {
		return (
			<div className="form-group">
				<label className="control-label" htmlFor={this.props.name}>{this.props.label}</label>
				<textarea type="text" className="form-control section-name" rows={this.props.rows}
					   name={this.props.name} value={this.props.value} onChange={this.props.onChange}
					   onKeyDown={this.onKeyDown} />
			</div>
		);
	}
});

var Alert = React.createClass({displayName: "Alert",
	propTypes: {
		type: React.PropTypes.oneOf(['success', 'info', 'warning', 'danger'])
	},
	getDefaultProps: function() {
		return {"type": "info"};
	},
	render: function() {
		if (this.props.children.length == 0) {
			return null;
		}
		return <div className={"alert alert-"+this.props.type} role="alert">{this.props.children}</div>;
	}
});

function getParameterByName(name) {
	name = name.replace(/[\[]/, "\\[").replace(/[\]]/, "\\]");
	var regex = new RegExp("[\\?&]" + name + "=([^&#]*)"),
		results = regex.exec(location.search);
	return results == null ? "" : decodeURIComponent(results[1].replace(/\+/g, " "));
}

function ancestorWithClass(el, className) {
	while (el != document && !el.classList.contains(className)) {
		el = el.parentNode;
	}
	if (el == document) {
		el = null;
	}
	return el;
}

if (!Array.prototype.filter) {
	Array.prototype.filter = function(fun /*, thisArg */) {
		"use strict";

		if (this === void 0 || this === null) {
			throw new TypeError();
		}

		var t = Object(this);
		var len = t.length >>> 0;
		if (typeof fun !== "function") {
			throw new TypeError();
		}

		var res = [];
		var thisArg = arguments.length >= 2 ? arguments[1] : void 0;
		for (var i = 0; i < len; i++) {
			if (i in t) {
				var val = t[i];

				// NOTE: Technically this should Object.defineProperty at
				//       the next index, as push can be affected by
				//       properties on Object.prototype and Array.prototype.
				//       But that method's new, and collisions should be
				//       rare, so use the more-compatible alternative.
				if (fun.call(thisArg, val, i, t)) {
					res.push(val);
				}
			}
		}

		return res;
	};
}
