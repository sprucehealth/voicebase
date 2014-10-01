/** @jsx React.DOM */

var phoneNumberPlaceholder = "###-###-####";

var API = {
	// cb is function(success: bool, data: object, jqXHR: jqXHR)
	ajax: function(params, cb) {
		params.success = function(data) {
			cb(true, data, null);
		}
		params.error = function(jqXHR) {
			cb(false, null, jqXHR);
		}
		jQuery.ajax(params);
	},
	updateCellNumber: function(number, cb) {
		this.ajax({
			type: "POST",
			contentType: "application/json",
			url: "/doctor-register/cell-verify",
			data: JSON.stringify({number: number}),
			dataType: "json"
		}, cb);
	},
	verifyCellNumber: function(number, code, cb) {
		this.ajax({
			type: "POST",
			contentType: "application/json",
			url: "/doctor-register/cell-verify",
			data: JSON.stringify({number: number, code: code}),
			dataType: "json"
		}, cb);
	},
	parseError: function(jqXHR) {
		var err;
		try {
			err = JSON.parse(jqXHR.responseText)
		} catch(e) {
			err = {error: jqXHR.responseText};
		}
		return err;
	}
};

var CellVerifyStep = React.createClass({displayName: "CellVerifyStep",
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
		return false;
	},
	onSubmit: function(e) {
		e.preventDefault();
		this.setState({
			error: null,
			busy: true
		});
		API.verifyCellNumber(this.props.number, this.state.code, function(success, data, jqXHR) {
			if (this.isMounted()) {
				if (!success) {
					this.setState({
						error: API.parseError(jqXHR),
						busy: false
					});
					return;
				}
				this.setState({busy: false});
				this.props.onSuccess();
			}
		}.bind(this));
		return false;
	},
	onCancel: function(e) {
		e.preventDefault();
		this.props.onCancel();
		return false;
	},
	render: function() {
		return (
			<div>
				<form method="POST" action="" role="form" className="form-onboard" onSubmit={this.onSubmit}>
					<p><strong>Verification code sent to {this.props.number}</strong></p>

					{this.state.error ? <Alert type="danger">{this.state.error}</Alert> : null}
					<FormInput required={true} label="Please enter the verification code you will receive" onChange={this.onChange} value={this.state.code} />
					<div className="text-center">
						<button type="submit" className="btn btn-default" onClick={this.onCancel}>Change Number</button>
						&nbsp;<button type="submit" className="btn btn-primary">Verify {this.busy ? <LoadingAnimation /> : null}</button>
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
		return false;
	},
	onSubmit: function(e) {
		e.preventDefault();
		this.setState({
			error: null,
			busy: true
		});
		API.updateCellNumber(this.state.number, function(success, data, jqXHR) {
			if (this.isMounted()) {
				if (!success) {
					this.setState({
						error: API.parseError(jqXHR),
						busy: false
					});
					return;
				}
				this.setState({busy: false});
				this.props.onSuccess(this.state.number);
			}
		}.bind(this));
		return false;
	},
	render: function() {
		return (
			<div>
				<form method="POST" action="" role="form" className="form-onboard" onSubmit={this.onSubmit}>
					{this.state.error ? <Alert type="danger">{this.state.error}</Alert> : null}
					<FormInput required={true} placeholder={phoneNumberPlaceholder} onChange={this.onChange} value={this.state.number} />
					<button type="submit" className="btn btn-primary center-block">Send Verification Text Message {this.busy ? <LoadingAnimation /> : null}</button>
				</form>
			</div>
		);
	}
});

var FormInput = React.createClass({displayName: "FormInput",
	propTypes: {
		type: React.PropTypes.string,
		name: React.PropTypes.string,
		label: React.PropTypes.renderable,
		value: React.PropTypes.string,
		placeholder: React.PropTypes.string,
		required: React.PropTypes.bool,
		onChange: React.PropTypes.func,
		onKeyDown: React.PropTypes.func
	},
	getDefaultProps: function() {
		return {
			type: "text",
			required: false
		}
	},
	render: function() {
		return (
			<div className="form-group">
				{this.props.label ? <label className="control-label" htmlFor={this.props.name}>{this.props.label}</label> : null}
				<input required={this.props.required ? "true" : null} type={this.props.type} className="form-control section-name"
					placeholder={this.props.placeholder} name={this.props.name} value={this.props.value}
					onKeyDown={this.props.onKeyDown} onChange={this.props.onChange} />
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

var LoadingAnimation = React.createClass({displayName: "LoadingAnimation",
	render: function() {
		return <img src={staticURL("/img/loading.gif")} />;
	}
});
