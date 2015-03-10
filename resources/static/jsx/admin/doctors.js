/** @jsx React.DOM */

var AdminAPI = require("./api.js");
var Forms = require("../forms.js");
var Modals = require("../modals.js");
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

						{this.state.results ? <DoctorSearchResults
							router={this.props.router}
							results={this.state.results} />
						: null}
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
			},
			{
				id: "eligibility",
				url: "eligibility",
				name: "State & Pathway Eligibility"
			},
			{
				id: "favorite_treatment_plans",
				url: "favorite_treatment_plans",
				name: "Favorite Treatment Plans"
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
		eligibility: function() {
			return <DoctorEligibilityPage router={this.props.router} doctor={this.state.doctor} />;
		},
		favorite_treatment_plans: function() {
			return <DoctorFTPPage router={this.props.router} doctor={this.state.doctor} />;
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

var DoctorFTPPage = React.createClass({displayName: "DoctorFTPPage",
	mixins: [Routing.RouterNavigateMixin],
	getInitialState: function() {
		return {
			ftps: [],
			ftp_fetch_error: null
		};
	},
	componentWillMount: function() {
		document.title = this.props.doctor.short_display_name + " | Doctors | Favorite Treatment Plans";
		AdminAPI.doctorFavoriteTreatmentPlans(this.props.doctor.id, function(success, data, error) {
			if (this.isMounted()) {
				if (!success) {
					this.setState({ftp_fetch_error: error.message});
					return;
				}
				this.setState({
					ftps: data.favorite_treatment_plans,
					ftp_fetch_error: null
				});
			}
		}.bind(this));
	},
	render: function() {
		content = []
		for(pathway in this.state.ftps) {
			for(var i = 0; i < this.state.ftps[pathway].length; ++i){
				this.state.ftps[pathway].sort(function(a, b){ 
					if(a.name < b.name) return -1
					if(a.name > b.name) return 1
					return 0
				})
				content.push(<tr>
											<td>{pathway}</td>
											<td>
												<a href={"/admin/treatment_plan/favorite/" + this.state.ftps[pathway][i].id + "/info"} onClick={this.onNavigate}>
													{this.state.ftps[pathway][i].name}
												</a>
											</td>
										 </tr>)
			}
		}
		return (
			<div className="container" style={{marginTop: 10}}>
				<div className="row">
					<div className="col-sm-12 col-md-12 col-lg-9">
						<h2>{this.props.doctor.short_display_name + " :: "}Favorite Treatment Plans</h2>
					</div>
				</div>

				<div className="row">
					<div className="col-md-12">
						<div className="text-center">
							{this.state.busy ? <Utils.LoadingAnimation /> : null}
							{this.state.ftp_fetch_error ? <Utils.Alert type="danger">{this.state.ftp_fetch_error}</Utils.Alert> : null}
						</div>
					</div>
				</div>

				<div className="row">
					<div className="col-md-12">
						<table className="table">
							<thead>
								<tr>
									<th>Pathway</th>
									<th>Name</th>
								</tr>
							</thead>
							<tbody>
								{content}
							</tbody>
						</table>
					</div>
				</div>
			</div>
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
						{Perms.has(Perms.DoctorsEdit) ?
							<button className="btn btn-default" data-toggle="modal" data-target="#avatarUpdateModal-thumbnail">
								Update
							</button>
						: null}
					</div>
					<div className="col-sm-6">
						<img src={this.state.profileImageURL["hero"]} className="doctor-thumbnail" />
						<br />
						Hero
						<br />
						{Perms.has(Perms.DoctorsEdit) ?
							<button className="btn btn-default" data-toggle="modal" data-target="#avatarUpdateModal-hero">
								Update
							</button>
						: null}
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

var LicenseStatuses = [
	{name: "ACTIVE", value: "ACTIVE"},
	{name: "INACTIVE", value: "INACTIVE"},
	{name: "TEMPORARY", value: "TEMPORARY"},
	{name: "PENDING", value: "PENDING"},
];

var DoctorLicensesPage = React.createClass({displayName: "DoctorLicensesPage",
	mixins: [Routing.RouterNavigateMixin],
	getInitialState: function() {
		return {
			licenses: [],
			error: null,
			editing: false,
		};
	},
	componentWillMount: function() {
		AdminAPI.medicalLicenses(this.props.doctor.id, function(success, data, error) {
			if (this.isMounted()) {
				if (success) {
					this.setState({licenses: data.licenses || [], error: null});
				} else {
					this.setState({error: "Failed to query licenses: " + error.message});
				}
			}
		}.bind(this));
	},
	onAdd: function(e) {
		e.preventDefault();
		this.state.licenses.push({
			state: "",
			number: "",
			status: "ACTIVE",
			expiration: ""
		})
		this.setState({
			licenses: this.state.licenses,
		});
	},
	onRemove: function(index, e) {
		e.preventDefault();
		for(var i = index; i < this.state.licenses.length-1; i++) {
			this.state.licenses[i] = this.state.licenses[i+1];
		}
		this.state.licenses.pop();
		this.setState({licenses: this.state.licenses});
	},
	onEdit: function(e) {
		e.preventDefault();
		this.setState({editing: true});
	},
	onSave: function(e) {
		e.preventDefault();
		var states = {};
		for(var i = 0; i < this.state.licenses.length; i++) {
			var l = this.state.licenses[i];
			if (l.state == "") {
				this.setState({error: "State is required"});
				return;
			}
			if (states[l.state]) {
				this.setState({error: "Can only have one license per state"});
				return;
			}
			states[l.state] = true;
		}
		AdminAPI.updateMedicalLicenses(this.props.doctor.id, this.state.licenses, function(success, data, error) {
			if (this.isMounted()) {
				if (success) {
					this.setState({licenses: data.licenses || [], error: null, editing: false});
				} else {
					this.setState({error: "Failed to update licenses: " + error.message});
				}
			}
		}.bind(this));
	},
	onCancel: function(e) {
		e.preventDefault();
		this.setState({editing: false});
		this.componentWillMount();
	},
	onChangeState: function(lic, e) {
		lic.state = e.target.value;
		this.setState({licenses: this.state.licenses});
	},
	onChangeLicenseNumber: function(lic, e) {
		lic.number = e.target.value;
		this.setState({licenses: this.state.licenses});
	},
	onChangeStatus: function(lic, e) {
		lic.status = e.target.value;
		this.setState({licenses: this.state.licenses});
	},
	onChangeExpiration: function(lic, e) {
		lic.expiration = e.target.value;
		this.setState({licenses: this.state.licenses});
	},
	createRow: function(lic, i) {
		if (this.state.editing) {
			return (
				<tr key={"license-"+i}>
					<td>
						<a href="#" onClick={this.onRemove.bind(this, i)}>
							<span className="glyphicon glyphicon-remove" style={{color:"red"}}></span>
						</a>
					</td>
					<td>
						<Forms.FormSelect
							value={lic.state}
							required={true}
							onChange={this.onChangeState.bind(this, lic)}
							opts={Utils.states} />
					</td>
					<td>
						<Forms.FormInput
							type="text"
							value={lic.number}
							required={true}
							onChange={this.onChangeLicenseNumber.bind(this, lic)} />
					</td>
					<td>
						<Forms.FormSelect
							value={lic.status}
							required={true}
							onChange={this.onChangeStatus.bind(this, lic)}
							opts={LicenseStatuses} />
					</td>
					<td>
						<Forms.FormInput
							type="date"
							value={lic.expiration}
							onChange={this.onChangeExpiration.bind(this, lic)} />
					</td>
				</tr>
			);
		}
		return (
			<tr key={"license-"+i}>
				<td>{lic.state}</td>
				<td>{lic.number}</td>
				<td>{lic.status}</td>
				<td>{lic.expiration}</td>
			</tr>
		);
	},
	render: function() {
		var table = (
			<table className="table">
				<thead>
					<tr>
						{this.state.editing ? <th style={{width:30}}></th> : null}
						<th>State</th>
						<th>Number</th>
						<th>Status</th>
						<th>Expiration</th>
					</tr>
				</thead>
				<tbody>
					{this.state.licenses.map(this.createRow)}
				</tbody>
			</table>);
		return (
			<div>
				<h2>{this.props.doctor.long_display_name} :: Medical Licenses</h2>
				{this.state.editing ?
					<div>
						<form onSubmit={this.onSave}>
							{table}
							{this.state.error ? <Utils.Alert type="danger">{this.state.error}</Utils.Alert> : null}
							<div className="pull-right">
								<button className="btn btn-default" onClick={this.onCancel}>Cancel</button>
								{" "}<button className="btn btn-primary" type="submit">Save</button>
							</div>
							<button className="btn btn-default" onClick={this.onAdd}>+</button>
						</form>
					</div>
				:
					<div>
						{table}
						{Perms.has(Perms.DoctorsEdit) ?
							<button className="pull-right btn btn-default" onClick={this.onEdit}>Edit</button>
						: null}
					</div>
				}
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
	onCancel: function(e) {
		e.preventDefault();
		if (!this.state.busy) {
			this.props.onDone();
		}
	},
	onSave: function(e) {
		e.preventDefault();
		if (this.state.busy) {
			return;
		}
		if (!this.state.modified) {
			this.props.onDone();
			return;
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
	},
	render: function() {
		var fields = [];
		for(var i = 0; i < doctorProfileFields.length; i++) {
			var f = doctorProfileFields[i];
			if (f.type == "textarea") {
				fields.push(
					<Forms.TextArea
						key={f.id}
						name={f.id}
						label={f.label}
						value={this.state.profile[f.id]}
						onChange={this.onChange} />);
			} else {
				fields.push(
					<Forms.FormInput
						key={f.id}
						name={f.id}
						type={f.type}
						label={f.label}
						value={this.state.profile[f.id]}
						onChange={this.onChange} />);
			}
		};
		var alert = null;
		if (this.state.error != "") {
			alert = <Utils.Alert type="danger">this.state.error</Utils.Alert>;
		}
		var spinner = null;
		if (this.state.busy) {
			spinner = <Utils.LoadingAnimation />;
		}
		return (
			<div>
				<h2>{this.props.doctor.long_display_name} :: Profile</h2>
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
			return <EditDoctorProfile
				doctor={this.props.doctor}
				profile={this.state.profile}
				onDone={this.doneEditing} />;
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
				<h2>{this.props.doctor.long_display_name} :: Profile</h2>
				{Perms.has(Perms.DoctorsEdit) ?
					<div className="pull-right">
						<button className="btn btn-default" onClick={this.edit}>Edit</button>
					</div>
				: null}
				{fields}
			</div>
		);
	}
});

var DoctorEligibilityPage = React.createClass({displayName: "DoctorEligibilityPage",
	mixins: [Routing.RouterNavigateMixin],
	getInitialState: function() {
		return {
			busy: false,
			error: null,
			mappings: [],
			pathwayMap: {},
			pathways: [],
			deletes: {},
			updates: {},
			creates: []
		};
	},
	componentWillMount: function() {
		this.setState({busy: true});
		AdminAPI.pathways(false, function(success, res, error) {
			if (this.isMounted()) {
				if (success) {
					var pathwayMap = {};
					for(var i = 0; i < res.pathways.length; i++) {
						var p = res.pathways[i];
						pathwayMap[p.tag] = p;
					}
					this.sortMappings(this.state.mappings, pathwayMap);
					this.setState({pathways: res.pathways, pathwayMap: pathwayMap});
				} else {
					this.setState({error: error.message});
				}
			}
		}.bind(this));
		AdminAPI.careProviderEligibility(this.props.doctor.id, function(success, data, error) {
			if (success) {
				if (this.isMounted()) {
					this.sortMappings(data.mappings, this.state.pathwayMap);
					this.setState({mappings: data.mappings, busy: false});
				}
			} else {
				this.setState({error: error.message});
			}
		}.bind(this));
	},
	sortMappings: function(mappings, pathwayMap) {
		mappings.sort(function(a, b) {
			if (a.state_code > b.state_code) {
				return 1;
			} else if (a.state_code < b.state_code) {
				return -1;
			}
			var aName = pathwayMap[a.pathway_tag] || a.pathway_tag;
			var bName = pathwayMap[b.pathway_tag] || b.pathway_tag;
			if (aName > bName) {
				return 1;
			} else if (aName < bName) {
				return -1;
			}
			return 0;
		});
	},
	findMapping: function(id) {
		for(var i = 0; i < this.state.mappings.length; i++) {
			var m = this.state.mappings[i];
			if (m.id == id) {
				return m;
			}
		}
		return null;
	},
	getUpdate: function(mappingID) {
		var update = this.state.updates[mappingID];
		if (typeof update == "undefined") {
			update = {notify: null, unavailable: null};
			this.state.updates[mappingID] = update;
		}
		return update;
	},
	handleToggleNotify: function(mappingID, e) {
		e.preventDefault();
		var existing = this.findMapping(mappingID);
		if (existing == null) {
			this.setState({error: "Consistency error. Please refresh the page."});
			return;
		}
		var upd = this.getUpdate(mappingID);
		if (upd.notify != null) {
			upd.notify = null;
		} else {
			upd.notify = !existing.notify;
		}
		this.setState({updates: this.state.updates});
	},
	handleToggleAvailability: function(mappingID, e) {
		e.preventDefault();
		var existing = this.findMapping(mappingID);
		if (existing == null) {
			this.setState({error: "Consistency error. Please refresh the page."});
			return;
		}
		var upd = this.getUpdate(mappingID);
		if (upd.unavailable != null) {
			upd.unavailable = null;
		} else {
			upd.unavailable = !existing.unavailable;
		}
		this.setState({updates: this.state.updates});
	},
	handleDelete: function(mappingID, e) {
		e.preventDefault();
		this.state.deletes[mappingID] = !(this.state.deletes[mappingID] || false);
		this.setState({deletes: this.state.deletes});
	},
	handleCancel: function(e) {
		e.preventDefault();
		this.reset();
	},
	reset: function() {
		this.setState({
			deletes: {},
			updates: {},
			creates: []
		});
	},
	handleSave: function(e) {
		e.preventDefault();
		this.setState({error: null, busy: true});
		var patch = {
			"delete": [],
			"create": [],
			"update": []
		};
		for(var id in this.state.deletes) {
			if (this.state.deletes[id]) {
				patch.delete.push(id);
			}
		}
		this.state.creates.forEach(function(m) {
			patch.create.push({
				"state_code": m.state_code,
				"pathway_tag": m.pathway_tag,
				"notify": m.notify,
				"unavailable": m.unavailable
			});
		})
		for(var id in this.state.updates) {
			var m = this.state.updates[id];
			if (m.notify != null || m.unavailable != null) {
				patch.update.push({
					"id": id,
					"notify": m.notify,
					"unavailable": m.unavailable
				});
			}
		}
		AdminAPI.updateCareProviderEligiblity(this.props.doctor.id, patch, function(success, data, error) {
			if (success) {
				if (this.isMounted()) {
					this.reset();
					this.sortMappings(data.mappings, this.state.pathwayMap);
					this.setState({
						mappings: data.mappings,
						busy: false});
				}
			} else {
				this.setState({error: error.message});
			}
		}.bind(this));
	},
	handleAdd: function(mapping) {
		this.state.creates.push(mapping);
		this.setState({creates: this.state.creates});
	},
	handleCancelCreate: function(index, e) {
		e.preventDefault();
		for(var i = index; i < this.state.creates.length-1; i++) {
			this.state.creates[i] = this.state.creates[i+1];
		}
		this.state.creates.pop();
		this.setState({creates: this.state.creates});
	},
	render: function() {
		return (
			<div>
				{Perms.has(Perms.DoctorsEdit) ?
					<span>
						<AddMappingModal onSuccess={this.handleAdd} pathways={this.state.pathways} />
						<div className="pull-right"><button className="btn btn-default" data-toggle="modal" data-target="#add-mapping-modal">+</button></div>
					</span>
				: null}

				<h2>
					{this.props.doctor.long_display_name} :: Profile
					{" "}{this.state.busy ? <Utils.LoadingAnimation /> : null}
				</h2>

				<table className="table">
				<thead>
					<tr>
						<th>State</th>
						<th>Pathway</th>
						<th>Notify</th>
						<th>Availability</th>
						{Perms.has(Perms.DoctorsEdit) ? <th></th> : null}
					</tr>
				</thead>
				<tbody>
					{this.state.mappings.map(function(m) {
						var p = this.state.pathwayMap[m.pathway_tag];
						var update = this.getUpdate(m.id);
						return (
							<tr
								key={"provider-" + m.id}
								style={
									this.state.deletes[m.id] === true ? {
										textDecoration: "line-through",
										backgroundColor: "#ffa0a0"
									} : {}}
							>
								<td>{m.state_code}</td>
								<td>{p ? p.name : m.pathway_tag}</td>
								<td style={update.notify != null ? {backgroundColor: "#a0a0ff"} : {}}>
									{(update.notify != null ? update.notify : m.notify) ? "YES" : "NO"}
									{" "}{Perms.has(Perms.DoctorsEdit) ?
										<span>[<a href="#" onClick={this.handleToggleNotify.bind(this, m.id)}>toggle</a>]</span>
									: null}
								</td>
								<td style={update.unavailable != null ? {backgroundColor: "#a0a0ff"} : {}}>
									{(update.unavailable != null ? update.unavailable : m.unavailable) ? "UNAVAILABLE" : "AVAILABLE"}
									{" "}{Perms.has(Perms.DoctorsEdit) ?
										<span>[<a href="#" onClick={this.handleToggleAvailability.bind(this, m.id)}>toggle</a>]</span>
									: null}
								</td>
								{Perms.has(Perms.DoctorsEdit) ?
									<td>
										<a href="#" onClick={this.handleDelete.bind(this, m.id)}>
											<span className="glyphicon glyphicon-remove" style={{color:"red"}}></span>
										</a>
									</td>
								: null}
							</tr>
						);
					}.bind(this))}
					{this.state.creates.map(function(m, index) {
						var p = this.state.pathwayMap[m.pathway_tag];
						return (
							<tr key={"new-pathway-" + index} style={{backgroundColor: "#a0ffa0"}}>
								<td>{m.state_code}</td>
								<td>{p ? p.name : m.pathway_tag}</td>
								<td>{m.notify ? "YES" : "NO"}</td>
								<td>{m.unavailable ? "UNAVAILABLE" : "AVAILABLE"}</td>
								{Perms.has(Perms.DoctorsEdit) ?
									<td>
										<a href="#" onClick={this.handleCancelCreate.bind(this, index)}>
											<span className="glyphicon glyphicon-remove" style={{color:"red"}}></span>
										</a>
									</td>
								: null}
							</tr>
						);
					}.bind(this))}
				</tbody>
				</table>

				<div className="text-center">
					{this.state.error ? <Utils.Alert type="danger">{this.state.error}</Utils.Alert> : null}
				</div>

				{Perms.has(Perms.DoctorsEdit) ?
					<div className="text-right">
						<button className="btn btn-default" onClick={this.handleCancel}>Cancel</button>
						{" "}<button className="btn btn-primary" onClick={this.handleSave}>Save</button>
					</div>
				: null}
			</div>
		);
	}
});


var AddMappingModal = React.createClass({displayName: "AddMappingModal",
	getInitialState: function() {
		return this.stateForProps(this.props);
	},
	stateForProps: function(props) {
		return {
			error: "",
			busy: false,
			state: "",
			pathwayTag: "",
			notify: false,
			unavailable: false
		}
	},
	componentWillReceiveProps: function(nextProps) {
		this.setState(this.stateForProps(nextProps));
	},
	onChangeState: function(e) {
		e.preventDefault();
		this.setState({error: "", state: e.target.value});
	},
	onChangePathway: function(e) {
		e.preventDefault();
		this.setState({error: "", pathwayTag: e.target.value});
	},
	onChangeNotify: function(e, value) {
		this.setState({error: "", notify: value});
	},
	onChangeUnvailable: function(e, value) {
		this.setState({error: "", unavailable: value});
	},
	onAdd: function(e) {
		if (!this.state.state) {
			this.setState({error: "state is required"});
			return true;
		}
		if (!this.state.pathwayTag) {
			this.setState({error: "pathway is required"});
			return true;
		}
		var mapping = {
			state_code: this.state.state,
			pathway_tag: this.state.pathwayTag,
			notify: this.state.notify,
			unavailable: this.state.unavailable
		};
		this.setState({
			busy: true,
			error: "",
			state: "",
			pathwayTag: "",
			notify: false,
			unavailable: false
		});
		this.props.onSuccess(mapping);
		return false;
	},
	render: function() {
		var pathwayOpts = [
			{name: "Select a Pathway", value: ""}
		];
		this.props.pathways.forEach(function(p) {
			pathwayOpts.push({name: p.name, value: p.tag});
		});
		return (
			<Modals.ModalForm id="add-mapping-modal" title="Add Pathway"
				cancelButtonTitle="Cancel" submitButtonTitle="Add"
				onSubmit={this.onAdd}>

				{this.state.error ? <Utils.Alert type="danger">{this.state.error}</Utils.Alert> : null}

				<Forms.FormSelect
					label = "State"
					value = {this.state.state}
					required = {true}
					onChange = {this.onChangeState}
					opts = {Utils.states} />
				<Forms.FormSelect
					label = "Pathway"
					value = {this.state.pathwayTag}
					required = {true}
					onChange = {this.onChangePathway}
					opts = {pathwayOpts} />
				<Forms.Checkbox
					label = "Notify"
					checked = {this.state.notify}
					onChange = {this.onChangeNotify} />
				<Forms.Checkbox
					label = "Unavailable"
					checked = {this.state.unavailable}
					onChange = {this.onChangeUnvailable} />
			</Modals.ModalForm>
		);
	}
});
