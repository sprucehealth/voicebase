/** @jsx React.DOM */

var AdminAPI = require("./api.js");
var Forms = require("../forms.js");
var Routing = require("../routing.js");
var Utils = require("../utils.js");

module.exports = {
	CareProviderStatePathwayMappings: React.createClass({displayName: "CareProviderStatePathwayMappings",
		mixins: [Routing.RouterNavigateMixin],
		getInitialState: function() {
			return {
				error: null,
				busy: false,
				stateCode: "",
				pathwayTag: "",
				pathways: {}
			};
		},
		componentWillMount: function() {
			AdminAPI.pathways(false, function(success, res, error) {
				if (this.isMounted()) {
					if (success) {
						var pathways = {};
						for(var i = 0; i < res.pathways.length; i++) {
							var p = res.pathways[i];
							pathways[p.tag] = p;
						}
						this.setState({pathways: pathways});
					} else {
						this.setState({error: error.message});
					}
				}
			}.bind(this));
		},
		handleChangeState: function(e) {
			e.preventDefault();
			this.setState({stateCode: e.target.value});
		},
		handleChangePathway: function(e) {
			e.preventDefault();
			this.setState({pathwayTag: e.target.value});
		},
		render: function() {
			var pathwayOpt = [{name: "Select Pathway", value: ""}];
			for(var id in this.state.pathways) {
				var p = this.state.pathways[id];
				pathwayOpt.push({name: p.name, value: p.tag});
			}
			var content;
			if (this.state.pathwayTag != "" || this.state.stateCode != "") {
				content = <StatePathwayMappings
					pathwayTag = {this.state.pathwayTag}
					stateCode = {this.state.stateCode}
					pathways = {this.state.pathways}
					router = {this.props.router} />
			} else {
				content = <StatePathwayMappingsSummary
					pathways = {this.state.pathways}
					onClickState = {function(stateCode) { this.setState({stateCode: stateCode}); }.bind(this)}
					onClickPathway = {function(pathwayTag) { this.setState({pathwayTag: pathwayTag}); }.bind(this)} />
			}
			return (
				<div className="container" style={{marginTop: 10}}>
					<div className="row">
						<div className="col-md-6">
							<Forms.FormSelect
								name = "state"
								label = "State"
								value = {this.state.stateCode}
								required = {true}
								onChange = {this.handleChangeState}
								opts = {Utils.states} />
						</div>
						<div className="col-md-6">
							<Forms.FormSelect
								name = "pathway"
								label = "Pathway"
								value = {this.state.pathwayTag}
								required = {true}
								onChange = {this.handleChangePathway}
								opts = {pathwayOpt} />
						</div>
					</div>

					<div className="row">
						<div className="col-md-12">
							<div className="text-center">
								{this.state.busy ? <Utils.LoadingAnimation /> : null}
								{this.state.error ? <Utils.Alert type="danger">{this.state.error}</Utils.Alert> : null}
							</div>
						</div>
					</div>

					<div className="row">
						<div className="col-md-12">
							{content}
						</div>
					</div>
				</div>
			);
		}
	})
};

var StatePathwayMappingsSummary = React.createClass({displayName: "StatePathwayMappingsSummary",
	mixins: [Routing.RouterNavigateMixin],
	propTypes: {
		pathways: React.PropTypes.object.isRequired,
		onClickState: React.PropTypes.func,
		onClickPathway: React.PropTypes.func
	},
	getInitialState: function() {
		return {
			error: null,
			busy: true,
			summary: []
		};
	},
	componentWillMount: function() {
		this.setState({busy: true, error: null});
		AdminAPI.careProviderStatePathwayMappingsSummary(function(success, res, error) {
			if (this.isMounted()) {
				if (success) {
					res.summary.sort(function(a, b) {
						if (a.state_code < b.state_code) {
							return -1;
						} else if (a.state_code > b.state_code) {
							return 1;
						}
						var aName = this.props.pathways[a.pathway_tag] || a.pathway_tag;
						var bName = this.props.pathways[b.pathway_tag] || b.pathway_tag;
						if (aName < bName) {
							return -1;
						} else if (aName > bName) {
							return 1;
						}
						return 0;
					}.bind(this));
					this.setState({busy: false, summary: res.summary});
				} else {
					this.setState({busy: false, error: error.message});
				}
			}
		}.bind(this));
	},
	onClickState: function(stateCode, e) {
		e.preventDefault();
		this.props.onClickState(stateCode);
	},
	onClickPathway: function(pathwayTag, e) {
		e.preventDefault();
		this.props.onClickPathway(pathwayTag);
	},
	render: function() {
		return (
			<div>
				<div className="text-center">
					{this.state.busy ? <Utils.LoadingAnimation /> : null}
					{this.state.error ? <Utils.Alert type="danger">{this.state.error}</Utils.Alert> : null}
				</div>
				<table className="table">
				<thead>
					<tr>
						<th>State</th>
						<th>Pathway</th>
						<th>Number of Doctors</th>
					</tr>
				</thead>
				<tbody>
					{this.state.summary.map(function(m) {
						var p = this.props.pathways[m.pathway_tag];
						return (
							<tr key={"summary-" + m.state_code + "-" + m.pathway_tag}>
								{this.props.onClickState ?
									<td><a href="#" onClick={this.onClickState.bind(this, m.state_code)}>{m.state_code}</a></td>
								:
									<td>{m.state_code}</td>}
								{this.props.onClickPathway ?
									<td><a href="#" onClick={this.onClickPathway.bind(this, m.pathway_tag)}>{p ? p.name : m.pathway_tag}</a></td>
								:
									<td>{p ? p.name : m.pathway_tag}</td>}
								<td>{m.doctor_count}</td>
							</tr>
						);
					}.bind(this))}
				</tbody>
				</table>
			</div>
		);
	}
});

var StatePathwayMappings = React.createClass({displayName: "StatePathwayMappings",
	mixins: [Routing.RouterNavigateMixin],
	propTypes: {
		router: React.PropTypes.object.isRequired,
		pathways: React.PropTypes.object.isRequired,
		stateCode: React.PropTypes.string,
		pathwayTag: React.PropTypes.string
	},
	getInitialState: function() {
		return {
			error: null,
			busy: false,
			mappings: []
		};
	},
	componentWillMount: function() {
		this.fetchMappings(this.props.stateCode, this.props.pathwayTag);
	},
	componentWillReceiveProps: function(nextProps) {
		this.fetchMappings(nextProps.stateCode, nextProps.pathwayTag);
	},
	fetchMappings: function(stateCode, pathwayTag) {
		this.setState({busy: true, error: null});
		AdminAPI.careProviderStatePathwayMappings({state: stateCode, pathwayTag: pathwayTag},
			function(success, res, error) {
				if (this.isMounted()) {
					if (success) {
						this.setState({busy: false, mappings: res.mappings});
					} else {
						this.setState({busy: false, error: error.message});
					}
				}
			}.bind(this));
	},
	render: function() {
		return (
			<div>
				<div className="text-center">
					{this.state.busy ? <Utils.LoadingAnimation /> : null}
					{this.state.error ? <Utils.Alert type="danger">{this.state.error}</Utils.Alert> : null}
				</div>
				<table className="table">
				<thead>
					<tr>
						<th>State</th>
						<th>Pathway</th>
						<th>Provider Role</th>
						<th>Provider Name</th>
						<th>Notify</th>
						<th>Availability</th>
					</tr>
				</thead>
				<tbody>
					{this.state.mappings.map(function(m) {
						var p = this.props.pathways[m.pathway_tag];
						return (
							<tr key={"mapping-" + m.id}>
								<td>{m.state_code}</td>
								<td>{p ? p.name : m.pathway_tag}</td>
								<td>{m.provider.role}</td>
								<td>
									<a href={"doctors/" + m.provider.id + "/eligibility"} onClick={this.onNavigate}>
										{m.short_display_name}
									</a>
								</td>
								<td>{m.notify ? "YES" : "NO"}</td>
								<td>{m.unavailable ? "UNAVAILABLE" : "AVAILABLE"}</td>
							</tr>
						);
					}.bind(this))}
				</tbody>
				</table>
			</div>
		);
	}
});
