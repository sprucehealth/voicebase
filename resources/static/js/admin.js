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
			type: "POST",
			contentType: "application/json",
			url: "/doctors/" + doctorID + "/profile",
			data: JSON.stringify(profile),
			dataType: "json"
		}, cb);
	},
	resourceGuide: function(id, cb) {
		this.ajax({
			type: "GET",
			url: "/guides/resources/" + id,
			dataType: "json"
		}, cb);
	},
	resourceGuidesList: function(sectionsOnly, cb) {
		var params = "";
		if (sectionsOnly) {
			params = "?sections_only=1"
		}
		this.ajax({
			type: "GET",
			url: "/guides/resources" + params,
			dataType: "json"
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
	updateResourceGuide: function(id, guide, cb) {
		this.ajax({
			type: "POST",
			contentType: "application/json",
			url: "/guides/resources/" + id,
			data: JSON.stringify(guide),
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
};

var RouterNavigateMixin = {
	navigate: function(path) {
		if (path.indexOf(Backbone.history.root) == 0) {
			path = path.substring(Backbone.history.root.length, path.length);
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
				<li key={item.id} className={active ? 'active' : ''}><a href={Backbone.history.root + item.url} onClick={this.onNavigate}>{item.name}</a></li>
			);
		}.bind(this));
		var rightMenuItems = this.props.rightItems.map(function(item) {
			var active = item.id == this.props.activeItem;
			return (
				<li key={item.id} className={active ? 'active' : ''}><a href={Backbone.history.root + item.url} onClick={this.onNavigate}>{item.name}</a></li>
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
						<a className="navbar-brand" href={Backbone.history.root} onClick={this.onNavigate}>{this.props.name}</a>
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
		var navItems = this.props.items.map(function(item) {
			var active = item.id == this.props.currentPage;
			return (
				<li key={item.id} className={active?"active":""}>
					<a href={item.url} onClick={this.onNavigate}>{item.name}</a>
				</li>
			);
		}.bind(this));
		return (
			<div>
				<div className="row">
					<div className="col-sm-3 col-md-2 sidebar">
						<ul className="nav nav-sidebar">
							{navItems}
						</ul>
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
			this.current = "dashboard"
		},
		"doctors": function() {
			this.current = "doctorSearch"
		},
		"doctors/:doctorID/:page": function(doctorID, page) {
			this.current = "doctor"
			this.params = {doctorID: doctorID, page: page}
		},
		"guides/:page": function(page) {
			this.current = "guides"
			this.params = {page: page}
		},
		"guides/:page/:id": function(page, guideID) {
			this.current = "guides"
			this.params = {page: page, guideID: guideID}
		}
	}
});

var Admin = React.createClass({displayName: "Admin",
	leftMenuItems: [
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
	],
	rightMenuItems: [],
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
				<TopNav router={this.props.router} leftItems={this.leftMenuItems} rightItems={this.rightMenuItems} activeItem={this.props.router.current} name="Admin" />
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
		this.onRefreshOnboardURL();
	},
	onRefreshOnboardURL: function() {
		AdminAPI.doctorOnboarding(function(success, res, jqXHR) {
			if (this.isMounted()) {
				if (!success) {
					console.log(jqXHR);
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

DoctorSearch = React.createClass({displayName: "DoctorSearch",
	mixins: [RouterNavigateMixin],
	getInitialState: function() {
		return {
			query: "",
			results: null
		};
	},
	componentWillMount: function() {
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
						console.log(jqXHR);
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
	menuItems: [
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
		}
	],
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
			attributes: {}
		};
	},
	componentWillMount: function() {
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
				<h2>{this.props.doctor.long_display_name}</h2>
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
			// TODO
			spinner = <img src="../img/loading.gif" />;
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

var Guides = React.createClass({displayName: "Guides",
	menuItems: [
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
	],
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
					data.layout_json = JSON.stringify(data.layout, null, 4);
					this.setState({guide: data});
				} else {
					alert("Failed to get resource guide");
				}
			}
		}.bind(this));
		AdminAPI.resourceGuidesList(true, function(success, data) {
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
		guide[e.target.name] = e.target.value;
		if (e.target.name == "layout_json") {
			try {
				var js = JSON.parse(e.target.value);
				this.state.guide.layout = js;
				this.setState({error: ""});
			} catch (err) {
				this.setState({error: "Invalid layout: " + err.message});
			};
		}
		this.setState({guide: guide});
		return false;
	},
	onSubmit: function() {
		AdminAPI.updateResourceGuide(this.props.guideID, this.state.guide, function(success, data, jqXHR) {
			if (this.isMounted()) {
				if (!success) {
					console.log(jqXHR);
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

				<form role="form" onSubmit={this.onSubmit}>
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
		AdminAPI.resourceGuidesList(false, function(success, data) {
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
			<div>{this.state.sections.map(createSection)}</div>
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
					this.setState({guide: data.guide, html: data.html});
				} else {
					alert("Failed to get rx guide");
				}
			}
		}.bind(this));
	},
	render: function() {
		return (
			<div dangerouslySetInnerHTML={{__html: this.state.html}}></div>
		);
	}
});

var RXGuideList = React.createClass({displayName: "RXGuideList",
	mixins: [RouterNavigateMixin],
	getInitialState: function() {
		return {"guides": []}
	},
	componentWillMount: function() {
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
	render: function() {
		return (
			<div className="rx-guide-list">
				<h2>RX Guides</h2>
				{this.state.guides.map(function(guide) {
					return <div key={guide.NDC} className="rx-guide">
						<img src={guide.ImageURL} width="32" height="32" />
						&nbsp;
						<a href={"/admin/guides/rx/" + guide.NDC} onClick={this.onNavigate}>{guide.Name + " (" + guide.NDC + ")"}</a>
					</div>
				}.bind(this))}
			</div>
		);
	}
});

// Form fields

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
						return <option value={opt.value}>{opt.name}</option>
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
				<input type={this.props.type} className="form-control section-name"
				       name={this.props.name} value={this.props.value} onChange={this.props.onChange} />
			</div>
		);
	}
});

var TextArea = React.createClass({displayName: "TextArea",
	getDefaultProps: function() {
		return {rows: 5}
	},
	render: function() {
		return (
			<div className="form-group">
				<label className="control-label" htmlFor={this.props.name}>{this.props.label}</label>
				<textarea type="text" className="form-control section-name" rows={this.props.rows}
				       name={this.props.name} value={this.props.value} onChange={this.props.onChange} />
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
