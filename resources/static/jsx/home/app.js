/** @jsx React.DOM */

var Forms = require("../forms.js");
var Utils = require("../utils.js");

var States = [
	{name: "Select Your State", value: ""},
	{name: "Alabama", value: "AL"},
	{name: "Alaska", value: "AK"},
	{name: "Arizona", value: "AZ"},
	{name: "Arkansas", value: "AR"},
	{name: "California", value: "CA"},
	{name: "Colorado", value: "CO"},
	{name: "Connecticut", value: "CT"},
	{name: "Delaware", value: "DE"},
	{name: "Florida", value: "FL"},
	{name: "Georgia", value: "GA"},
	{name: "Hawaii", value: "HI"},
	{name: "Idaho", value: "ID"},
	{name: "Illinois", value: "IL"},
	{name: "Indiana", value: "IN"},
	{name: "Iowa", value: "IA"},
	{name: "Kansas", value: "KS"},
	{name: "Kentucky", value: "KY"},
	{name: "Louisiana", value: "LA"},
	{name: "Maine", value: "ME"},
	{name: "Maryland", value: "MD"},
	{name: "Massachusetts", value: "MA"},
	{name: "Michigan", value: "MI"},
	{name: "Minnesota", value: "MN"},
	{name: "Mississippi", value: "MS"},
	{name: "Missouri", value: "MO"},
	{name: "Montana", value: "MT"},
	{name: "Nebraska", value: "NE"},
	{name: "Nevada", value: "NV"},
	{name: "New Hampshire", value: "NH"},
	{name: "New Jersey", value: "NJ"},
	{name: "New Mexico", value: "NM"},
	{name: "New York", value: "NY"},
	{name: "North Carolina", value: "NC"},
	{name: "North Dakota", value: "ND"},
	{name: "Ohio", value: "OH"},
	{name: "Oklahoma", value: "OK"},
	{name: "Oregon", value: "OR"},
	{name: "Pennsylvania", value: "PA"},
	{name: "Rhode Island", value: "RI"},
	{name: "South Carolina", value: "SC"},
	{name: "South Dakota", value: "SD"},
	{name: "Tennessee", value: "TN"},
	{name: "Texas", value: "TX"},
	{name: "Utah", value: "UT"},
	{name: "Vermont", value: "VT"},
	{name: "Virginia", value: "VA"},
	{name: "Washington", value: "WA"},
	{name: "Washington, D.C.", value: "DC"},
	{name: "West Virginia", value: "WV"},
	{name: "Wisconsin", value: "WI"},
	{name: "Wyoming", value: "WY"}
];

var UnsupportedPlatforms = [
	{name: "Select Your Phone", value: ""},
	{name: "Android", value: "Android"},
	{name: "iPhone", value: "iPhone"}
];

var API = {
	// cb is function(success: bool, data: object, jqXHR: jqXHR)
	ajax: function(params, cb) {
		params.success = function(data) {
			cb(true, data, "", null);
		}
		params.error = function(jqXHR) {
			cb(false, null, API.parseError(jqXHR), jqXHR);
		}
		params.url = "/api" + params.url;
		jQuery.ajax(params);
	},
	parseError: function(jqXHR) {
		if (jqXHR.status == 0) {
			return {message: "Network request failed"};
		}
		var err;
		try {
			err = JSON.parse(jqXHR.responseText).error;
		} catch(e) {
			console.error(jqXHR.responseText);
			err = {message: "Unknown error"};
		}
		return err;
	},

	//

	recordForm: function(name, values, cb) {
		this.ajax({
			type: "POST",
			contentType: "application/json",
			url: "/forms/" + encodeURIComponent(name),
			data: JSON.stringify(values),
			dataType: "json"
		}, cb);
	}
};

function staticURL(path) {
	return Spruce.BaseStaticURL + path
}

window.NotifyMeComponent = React.createClass({displayName: "NotifyMeComponent",
	getInitialState: function() {
		return {
			email: "",
			state: "",
			platform: "",
			busy: false,
			error: null
		}
	},
	onSubmit: function(e) {
		e.preventDefault();
		if (this.state.busy) {
			return false;
		}
		this.setState({busy: true, error: null});
		API.recordForm("notify-me", {email: this.state.email, state: this.state.state, platform: this.state.platform},
			function(success, data, error) {
				if (this.isMounted()) {
					if (!success) {
						this.setState({busy: false, error: error});
						return;
					}
					this.setState({busy: false});
					$("#notify-me-modal").modal('hide');
				}
			}.bind(this));
		return false;
	},
	onChangeEmail: function(e) {
		e.preventDefault();
		this.setState({email: e.target.value});
		return false;
	},
	onChangeState: function(e) {
		e.preventDefault();
		this.setState({state: e.target.value});
		return false;
	},
	onChangePlatform: function(e) {
		e.preventDefault();
		this.setState({platform: e.target.value});
		return false;
	},
	render: function() {
		return (
			<form method="POST" action="#" onSubmit={this.onSubmit} className="text-center">
				<h3>Sign up to be notified when Spruce is available to you.</h3>
				<br />
				<Forms.FormInput placeholder="Your Email Address" value={this.state.email} type="email" required={true} onChange={this.onChangeEmail} />
				<div className="row">
					<div className="col-md-6">
						<Forms.FormSelect value={this.state.state} required={true} onChange={this.onChangeState} opts={States} />
					</div>
					<div className="col-md-6">
						<Forms.FormSelect value={this.state.platform} required={true} onChange={this.onChangePlatform} opts={UnsupportedPlatforms} />
					</div>
				</div>
				{this.state.error ? <Utils.Alert type="danger">{this.state.error}</Utils.Alert> : null}
				<button type="submit" className="btn btn-primary">Sign Up {this.state.busy ? <Utils.LoadingAnimation /> : null}</button>
			</form>
		);
	}
});

window.DoctorInterestComponent = React.createClass({displayName: "DoctorInterestComponent",
	getInitialState: function() {
		return {
			name: "",
			email: "",
			states: "",
			comment: "",
			busy: false,
			error: null
		}
	},
	onSubmit: function(e) {
		e.preventDefault();
		if (this.state.busy) {
			return false;
		}
		this.setState({busy: true, error: null});
		API.recordForm("doctor-interest", {
				name: this.state.name,
				email: this.state.email,
				states: this.state.states,
				comment: this.state.comment
			},
			function(success, data, error) {
				if (this.isMounted()) {
					if (!success) {
						this.setState({busy: false, error: error});
						return;
					}
					this.setState({busy: false});
					$("#doctor-interest-modal").modal('hide');
				}
			}.bind(this));
		return false;
	},
	onChangeName: function(e) {
		e.preventDefault();
		this.setState({name: e.target.value});
		return false;
	},
	onChangeEmail: function(e) {
		e.preventDefault();
		this.setState({email: e.target.value});
		return false;
	},
	onChangeStates: function(e) {
		e.preventDefault();
		this.setState({states: e.target.value});
		return false;
	},
	onChangeComment: function(e) {
		e.preventDefault();
		this.setState({comment: e.target.value});
		return false;
	},
	render: function() {
		return (
			<form method="POST" action="#" onSubmit={this.onSubmit} className="text-center">
				<h2>Get In Touch</h2>
				<p>Tell us a little bit about yourself and someone from Spruce will be in touch shortly.</p>
				<Forms.FormInput placeholder="Your Name" value={this.state.name} required={true} onChange={this.onChangeName} />
				<Forms.FormInput placeholder="Your Email Address" value={this.state.email} type="email" required={true} onChange={this.onChangeEmail} />
				<Forms.FormInput placeholder="States Where You're Licensed" value={this.state.states} required={true} onChange={this.onChangeStates} />
				<Forms.FormInput placeholder="Optional Comment" value={this.state.comment} onChange={this.onChangeComment} />
				{this.state.error ? <Utils.Alert type="danger">{this.state.error}</Utils.Alert> : null}
				<button type="submit" className="btn btn-primary">Submit {this.state.busy ? <Utils.LoadingAnimation /> : null}</button>
			</form>
		);
	}
});
