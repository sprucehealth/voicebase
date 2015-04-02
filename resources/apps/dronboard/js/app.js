/* @flow */

var API = require("./api.js");
var Forms = require("../../libs/forms.js");
var Utils = require("../../libs/utils.js");
var React = require("react");
window.React = React; // export for http://fb.me/react-devtools

var phoneNumberPlaceholder = "###-###-####";

window.CellVerifyStep = React.createClass({displayName: "CellVerifyStep",
	propTypes: {
		number: React.PropTypes.string,
		nextURL: React.PropTypes.string
	},
	getInitialState: function() {
		return {
			number: this.props.number || "",
			sent: false
		};
	},
	onCodeSent: function(number) {
		this.setState({number: number, sent: true});
	},
	onCancelVerify: function() {
		this.setState({sent: false});
	},
	onSuccess: function() {
		window.location.href = this.props.nextURL;
	},
	render: function() {
		if (this.state.sent) {
			return <CellVerify number={this.state.number} onSuccess={this.onSuccess} onCancel={this.onCancelVerify} />;
		}
		return <CellEntryForm number={this.state.number} onSuccess={this.onCodeSent} />;
	}
});

var CellVerify = React.createClass({displayName: "CellVerify",
	getInitialState: function() {
		return {
			code: "",
			error: null,
			busy: false
		};
	},
	onChange: function(e) {
		e.preventDefault();
		this.setState({code: e.target.value});
	},
	onSubmit: function(e) {
		e.preventDefault();
		this.setState({
			error: null,
			busy: true
		});
		API.verifyCellNumber(this.props.number, this.state.code, function(success, data, error, jqXHR) {
			if (this.isMounted()) {
				if (!success) {
					this.setState({
						error: error.message,
						busy: false
					});
					return;
				}
				this.setState({busy: false});
				this.props.onSuccess();
			}
		}.bind(this));
	},
	onCancel: function(e) {
		e.preventDefault();
		this.props.onCancel();
	},
	render: function() {
		return (
			<div>
				<form method="POST" action="" role="form" className="form-onboard" onSubmit={this.onSubmit}>
					<p><strong>Verification code sent to {this.props.number}</strong></p>

					{this.state.error ? <Utils.Alert type="danger">{this.state.error}</Utils.Alert> : null}
					<Forms.FormInput required={true} label="Please enter the verification code you will receive" onChange={this.onChange} value={this.state.code} />
					<div className="text-center">
						<button type="submit" className="btn btn-default" onClick={this.onCancel}>Change Number</button>
						&nbsp;<button type="submit" className="btn btn-primary">Verify {this.state.busy ? <Utils.LoadingAnimation /> : null}</button>
					</div>
				</form>
			</div>
		);
	}
});

var CellEntryForm = React.createClass({displayName: "CellEntryForm",
	getInitialState: function() {
		return {
			number: this.props.number || "",
			error: null,
			busy: false
		};
	},
	onChange: function(e) {
		e.preventDefault();
		this.setState({number: e.target.value});
	},
	onSubmit: function(e) {
		e.preventDefault();
		this.setState({
			error: null,
			busy: true
		});
		API.updateCellNumber(this.state.number, function(success, data, error, jqXHR) {
			if (this.isMounted()) {
				if (!success) {
					this.setState({
						error: error.message,
						busy: false
					});
					return;
				}
				this.setState({busy: false});
				this.props.onSuccess(this.state.number);
			}
		}.bind(this));
	},
	render: function() {
		return (
			<div>
				<form method="POST" action="" role="form" className="form-onboard" onSubmit={this.onSubmit}>
					{this.state.error ? <Utils.Alert type="danger">{this.state.error}</Utils.Alert> : null}
					<Forms.FormInput required={true} placeholder={phoneNumberPlaceholder} onChange={this.onChange} value={this.state.number} />
					<button type="submit" className="btn btn-primary center-block">Send Verification Text Message {this.state.busy ? <Utils.LoadingAnimation /> : null}</button>
				</form>
			</div>
		);
	}
});
