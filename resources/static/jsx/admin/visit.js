var AdminAPI = require("./api.js");
var Forms = require("../forms.js");
var Modals = require("../modals.js");
var Nav = require("../nav.js");
var Perms = require("./permissions.js");
var Routing = require("../routing.js");
var Utils = require("../utils.js");

module.exports = {
	Page: React.createClass({displayName: "VisitPage",
		menuItems: [[
			{
				id: "visit",
				url: "/admin/visit",
				name: "Overview"
			}
		]],
		getDefaultProps: function() {
			return {}
		},
		overview: function() {
			return <VisitOverviewPage router={this.props.router} />;
		},
		details: function() {
			return <VisitDetailsPage router={this.props.router} caseID={this.props.caseID} visitID={this.props.visitID} />;
		},
		render: function() {
			return (
				<div>
					<Nav.LeftNav router={this.props.router} items={this.menuItems} currentPage={this.props.page}>
						{this[this.props.page]()}
					</Nav.LeftNav>
				</div>
			);
		}
	})
};

var VisitOverviewPage = React.createClass({displayName: "VisitOverviewPage",
	mixins: [Routing.RouterNavigateMixin],
	getInitialState: function() {
		return {};
	},
	componentWillMount: function() {
		document.title = "Visit Overview";
	},
	render: function() {
		return (
			<div className="container" style={{marginTop: 10}}>
				<VisitSummaryList router={this.props.router} visitStatus="uncompleted" /> 
			</div>
		);
	}
});

var VisitSummaryList = React.createClass({ displayName: "VisitSummaryList",
	mixins: [Routing.RouterNavigateMixin],
	getInitialState: function() {
		return {
			busy: true,
			summaries: null
		};
	},
	componentWillMount: function() {
		AdminAPI.visitSummaries(this.props.visitStatus, function(success, data, error) {
			if (this.isMounted()) {
				if (!success) {
					this.setState({
						visitOverviewError: error.message,
						busy: false,
					});
					return;
				}
				this.setState({
					summaries: data.visit_summaries,
					visitOverviewError: null,
					busy: false,
				});
			}
		}.bind(this));
	},
	render: function() {
		var summaryCards = []
		if(this.state.summaries != null) {
			for(var i = 0; i < this.state.summaries.length; ++i){
				summaryCards.push(<VisitSummaryCard key={this.state.summaries[i].visit_id} router={this.props.router} summary={this.state.summaries[i]} />)
			}
		}
		return (
			<div className="container" style={{marginTop: 10}}>
			{
				this.state.busy ?
					<img src={Utils.staticURL("/img/loading.gif")} /> :
					this.state.visitOverviewError ? <Utils.Alert type="danger">{this.state.visitOverviewError}</Utils.Alert> : summaryCards
			}
			</div>
		);
	}
});

var VisitSummaryCard = React.createClass({ displayName: "VisitSummaryCard",
	mixins: [Routing.RouterNavigateMixin],
	getInitialState: function() {
		return {
			busy: true,
		};
	},
	componentWillMount: function() {
		this.setState({
			busy: false,
		});
	},
	render: function() {
		var visitSubmitted = new Date(0)
		var lockTaken = new Date(0)
		var currentEpoch = new Date().getTime() / 1000
		visitSubmitted.setUTCSeconds(this.props.summary.submission_epoch)
		lockTaken.setUTCSeconds(this.props.summary.lock_taken_epoch)
		return (
			<div className="card">
			<table className="table">
				<tbody>
					<tr>
						<td><div className="initials-circle">{this.props.summary.patient_initials}</div></td><td><div className="card-title">{this.props.summary.case_name}</div></td><td><div className="card-title">{this.props.summary.submission_state}</div></td>
					</tr>
					<tr>
						<td>Visit Submitted:</td><td>{visitSubmitted.toString()}</td><td>{Utils.timeSince(visitSubmitted.getTime()/1000, currentEpoch)}</td>
					</tr>
					<tr>
						<td>Doctor:</td><td>{this.props.summary.doctor_id != null ? <div className="success-text">{this.props.summary.doctor_with_lock}</div> : <div className="alert-text">Unassigned</div>}</td><td><strong>{this.props.summary.first_available ? "First Available" : "Doctor Selected"}</strong></td>
					</tr>
					{
						this.props.summary.doctor_id != null ?
						<tr>
							<td>Lock Acquired:</td><td>{lockTaken.toString()}</td><td>{Utils.timeSince(lockTaken.getTime()/1000, currentEpoch)}</td>
						</tr> : null
					}
					{
						this.props.summary.doctor_id != null ?
						<tr>
							<td>Lock Type:</td><td>{this.props.summary.lock_type == "TEMP" ? <div className="alert-text">this.props.summary.lock_type</div> : this.props.summary.lock_type}</td>
						</tr> : null
					}
					<tr>
						<td>Visit Type:</td><td>{this.props.summary.type}</td>
					</tr>
					<tr>
						<td>Internal Visit Status:</td><td>{this.props.summary.status}</td>
					</tr>
					<tr>
						<td>Visit ID:</td><td>{this.props.summary.visit_id}</td>
					</tr>
				</tbody>
			</table>
			</div>
		);
	}
});