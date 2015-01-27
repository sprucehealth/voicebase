/** @jsx React.DOM */

var AdminAPI = require("./api.js");
var Forms = require("../forms.js");
var Nav = require("../nav.js");
var Perms = require("./permissions.js");
var Routing = require("../routing.js");
var Utils = require("../utils.js");

module.exports = {
	DoctorSearch: React.createClass({displayName: "DoctorSearch",
		mixins: [Routing.RouterNavigateMixin],
		getInitialState: function() {
			return {
				query: "",
				busy: false,
				error: null,
				results: null
			};
		},
		componentWillMount: function() {
			document.title = "Search | Doctors | Spruce Admin";
			var q = Utils.getParameterByName("q");
			if (q != this.state.query) {
				this.setState({query: q});
				this.search(q);
			}

			// TODO: need to make sure the page that was navigated to is this one
			// this.navCallback = (function() {
			// 	var q = Utils.getParameterByName("q");
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
				this.setState({results: null});
			} else {
				this.setState({busy: true, error: null});
				AdminAPI.searchDoctors(q, function(success, res, error) {
					if (this.isMounted()) {
						if (!success) {
							this.setState({busy: false, error: error.message});
							return;
						}
						this.setState({busy: false, error: null, results: res.results || []});
					}
				}.bind(this));
			}
		},
		onSearchSubmit: function(e) {
			e.preventDefault();
			this.search(this.state.query);
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

					<div className="search-results">
						<div className="text-center">
							{this.state.busy ? <Utils.LoadingAnimation /> : null}
							{this.state.error ? <Utils.Alert type="danger">{this.state.error}</Utils.Alert> : null}
						</div>

						{this.state.results ? DoctorSearchResults({
							router: this.props.router,
							results: this.state.results}) : null}
					</div>
				</div>
			);
		}
	}),

	Doctor: React.createClass({displayName: "Doctor",
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
			}
		]],
		getInitialState: function() {
			return {
				doctor: null,
				error: null
			};
		},
		componentWillMount: function() {
			this.fetchDoctor();
		},
		fetchDoctor: function() {
			this.setState({error: null});
			AdminAPI.doctor(this.props.doctorID, function(success, data, error) {
				if (this.isMounted()) {
					if (!success) {
						this.setState({error: error.message});
						return;
					}

					var doctor = data.doctor;
					AdminAPI.account(doctor.account_id, function(success, data, error) {
						if (this.isMounted()) {
							if (!success) {
								this.setState({error: error.message});
								return;
							}
							doctor.account = data.account;
							document.title = doctor.short_display_name + " | Doctors | Spruce Admin";
							this.setState({doctor: doctor});
						}
					}.bind(this));
				}
			}.bind(this));
		},
		onAccountUpdate: function(account) {
			var doctor = this.state.doctor;
			doctor.account = account;
			this.setState({doctor: doctor});
		},
		info: function() {
			return <DoctorInfoPage router={this.props.router} doctor={this.state.doctor} onAccountUpdate={this.onAccountUpdate} />;
		},
		licenses: function() {
			return <DoctorLicensesPage router={this.props.router} doctor={this.state.doctor} />;
		},
		profile: function() {
			return <DoctorProfilePage router={this.props.router} doctor={this.state.doctor} />;
		},
		render: function() {
			return (
				<div>
					<Nav.LeftNav router={this.props.router} items={this.menuItems} currentPage={this.props.page}>
						{this.state.error ? <Utils.Alert type="danger">{this.state.error}</Utils.Alert> : null}
						{this.state.doctor == null ? <Utils.LoadingAnimation /> : this[this.props.page]()}
					</Nav.LeftNav>
				</div>
			);
		}
	})
};

var DoctorSearchResults = React.createClass({displayName: "DoctorSearchResult",
	mixins: [Routing.RouterNavigateMixin],
	render: function() {
		if (this.props.results.length == 0) {
			return (<div className="no-results text-center">No matching doctors found</div>);
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
			<div>{results}</div>
		);
	}
});

var DoctorSearchResult = React.createClass({displayName: "DoctorSearchResult",
	mixins: [Routing.RouterNavigateMixin],
	render: function() {
		return (
			<a href={"doctors/"+this.props.result.doctor_id+"/info"} onClick={this.onNavigate}>{"Dr. "+this.props.result.first_name+" "+this.props.result.last_name+" ("+this.props.result.email+")"}</a>
		);
	}
});

var DoctorInfoPage = React.createClass({displayName: "DoctorInfoPage",
	mixins: [Routing.RouterNavigateMixin],
	getInitialState: function() {
		return {
			updateAvatar: "",
			twoFactorError: null,
			attributes: {},
			attributesError: null,
			phoneNumbers: [],
			phoneNumbersError: null,
			profileImageURL: {
				"thumbnail": AdminAPI.doctorProfileImageURL(this.props.doctor.id, "thumbnail"),
				"hero": AdminAPI.doctorProfileImageURL(this.props.doctor.id, "hero")
			}
		};
	},
	componentWillMount: function() {
		document.title = this.props.doctor.short_display_name + " | Doctors | Spruce Admin";
		AdminAPI.doctorAttributes(this.props.doctor.id, function(success, data, error) {
			if (this.isMounted()) {
				if (!success) {
					this.setState({attributesError: error.message});
					return;
				}
				this.setState({attributes: data});
			}
		}.bind(this));
		AdminAPI.accountPhoneNumbers(this.props.doctor.account_id, function(success, data, error) {
			if (this.isMounted()) {
				if (!success) {
					this.setState({phoneNumbersError: error.message});
					return;
				}
				this.setState({phoneNumbers: data.numbers || []});
			}
		}.bind(this));
	},
	onUpdate: function() {
		// Change the profileImage URLs to force them to reload
		var v = Math.floor((Math.random() * 100000) + 1);
		this.setState({
			profileImageURL: {
				"thumbnail": AdminAPI.doctorProfileImageURL(this.props.doctor.id, "thumbnail")+"?v="+v,
				"hero": AdminAPI.doctorProfileImageURL(this.props.doctor.id, "hero")+"?v="+v
			}
		});
	},
	onTwoFactorToggle: function(e) {
		e.preventDefault();
		this.setState({twoFactorError: null});
		AdminAPI.updateAccount(this.props.doctor.account_id, {two_factor_enabled: !this.props.doctor.account.two_factor_enabled},
			function(success, data, error) {
				if (this.isMounted()) {
					if (!success) {
						this.setState({twoFactorError: error.message});
						return
					}
					this.props.onAccountUpdate(data.account);
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
				<DoctorUpdateProfileImageModal onUpdate={this.onUpdate} doctor={this.props.doctor} pImageType="thumbnail" />
				<DoctorUpdateProfileImageModal onUpdate={this.onUpdate} doctor={this.props.doctor} pImageType="hero" />
				<h2>{this.props.doctor.long_display_name}</h2>
				<h3>Profile Images</h3>
				<div className="row text-center">
					<div className="col-sm-6">
						<img src={this.state.profileImageURL["thumbnail"]} className="doctor-thumbnail" />
						<br />
						Thumbnail
						<br />
						<button className="btn btn-default" data-toggle="modal" data-target="#avatarUpdateModal-thumbnail">
						Update
						</button>
					</div>
					<div className="col-sm-6">
						<img src={this.state.profileImageURL["hero"]} className="doctor-thumbnail" />
						<br />
						Hero
						<br />
						<button className="btn btn-default" data-toggle="modal" data-target="#avatarUpdateModal-hero">
						Update
						</button>
					</div>
				</div>
				<h3>Two Factor Authentication</h3>
				{this.state.twoFactorError ? <Utils.Alert type="danger">{this.state.twoFactorError}</Utils.Alert> : null}
				<p>
					{this.props.doctor.account.two_factor_enabled ? "Enabled" : "Disabled"}
					&nbsp;[<a href="#" onClick={this.onTwoFactorToggle}>Toggle</a>]
				</p>
				<h3>Phone Numbers</h3>
				{this.state.phoneNumbersError ? <Utils.Alert type="danger">{this.state.phoneNumbersError}</Utils.Alert> : null}
				<table className="table">
					<thead>
						<tr>
							<th>Status</th>
							<th>Type</th>
							<th>Number</th>
							<th>Verified</th>
						</tr>
					</thead>
					<tbody>
						{this.state.phoneNumbers.map(function(num) {
							return (
								<tr key={num.phone}>
									<td>{num.status}</td>
									<td>{num.phone_type}</td>
									<td>{num.phone}</td>
									<td>{num.verified ? "Yes" : "No"}</td>
								</tr>
							);
						})}
					</tbody>
				</table>
				<h3>General Info</h3>
				{this.state.attributesError ? <Utils.Alert type="danger">{this.state.attributesError}</Utils.Alert> : null}
				<table className="table">
					<tbody>
						<tr>
							<td><strong>Email</strong></td>
							<td><a href={"mailto:"+this.props.doctor.email}>{this.props.doctor.email}</a></td>
						</tr>
						<tr>
							<td><strong>Registered</strong></td>
							<td>{this.props.doctor.account.registered}</td>
						</tr>
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

var DoctorUpdateProfileImageModal = React.createClass({displayName: "DoctorUpdateProfileImageModal",
	onSubmit: function(e) {
		e.preventDefault();
		var formData = new FormData(e.target);
		AdminAPI.updateDoctorProfileImage(this.props.doctor.id, this.props.pImageType, formData, function(success, data, error) {
			if (!success) {
				// TODO
				alert("Failed to upload profileImage: " + error.message);
				return;
			}
			$("#avatarUpdateModal-"+this.props.pImageType).modal('hide');
			this.props.onUpdate();
		}.bind(this));
	},
	render: function() {
		return (
			<div className="modal fade" id={"avatarUpdateModal-"+this.props.pImageType} role="dialog" tabIndex="-1">
				<div className="modal-dialog">
					<div className="modal-content">
						<form role="form" onSubmit={this.onSubmit}>
							<div className="modal-header">
								<button type="button" className="close" data-dismiss="modal"><span aria-hidden="true">&times;</span><span className="sr-only">Close</span></button>
								<h4 className="modal-title" id={"avatarUpdateModalTitle-"+this.props.pImageType}>Update {this.props.pImageType} Image</h4>
							</div>
							<div className="modal-body">
								<input required type="file" name="profile_image" />
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
	mixins: [Routing.RouterNavigateMixin],
	getInitialState: function() {
		return {licenses: []};
	},
	componentWillMount: function() {
		AdminAPI.medicalLicenses(this.props.doctor.id, function(success, licenses, error) {
			if (this.isMounted()) {
				if (success) {
					this.setState({licenses: licenses || []});
				} else {
					alert("Failed to get licenses: " + error.message);
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
	mixins: [Routing.RouterNavigateMixin],
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
		AdminAPI.updateCareProviderProfile(this.props.doctor.id, this.state.profile, function(success, data, error) {
			if (this.isMounted()) {
				this.setState({busy: false});
				if (success) {
					this.props.onDone();
				} else {
					this.setState({error: "Save failed: " + error.message});
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
				fields.push(Forms.TextArea({key: f.id, name: f.id, label: f.label, value: this.state.profile[f.id], onChange: this.onChange}));
			} else {
				fields.push(Forms.FormInput({key: f.id, name: f.id, type: f.type, label: f.label, value: this.state.profile[f.id], onChange: this.onChange}));
			}
		};
		var alert = null;
		if (this.state.error != "") {
			alert = Utils.Alert({"type": "danger"}, this.state.error);
		}
		var spinner = null;
		if (this.state.busy) {
			spinner = <Utils.LoadingAnimation />;
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
	mixins: [Routing.RouterNavigateMixin],
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
		AdminAPI.careProviderProfile(this.props.doctor.id, function(success, profile, error) {
			if (success) {
				if (this.isMounted()) {
					this.setState({profile: profile, busy: false});
				}
			} else {
				// TODO
				alert("Failed to get profile: " + error.message)
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
