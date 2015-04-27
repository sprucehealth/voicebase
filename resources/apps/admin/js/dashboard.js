/* @flow */

var AdminAPI = require("./api.js");
var Perms = require("./permissions.js");
var React = require("react");
var Routing = require("../../libs/routing.js");
var Utils = require("../../libs/utils.js");

module.exports = {
	Dashboard: React.createClass({displayName: "Dashboard",
		mixins: [Routing.RouterNavigateMixin],
		getInitialState: function() {
			return {onboardURL: ""}
		},
		componentWillMount: function() {
			document.title = "Dashboard | Spruce Admin";
			this.onRefreshOnboardURL(null);
		},
		onRefreshOnboardURL: function(e: ?Event): void {
			if (e) {
				e.preventDefault();
			}
			AdminAPI.doctorOnboarding(function(success, res, error) {
				if (this.isMounted()) {
					if (!success) {
						this.setState({onboardURL: "FAILED: " + error.message})
						return;
					}
					this.setState({onboardURL: res});
				}
			}.bind(this));
		},
		render: function(): any {
			return (
				<div>
					<div className="row">
						{Perms.has(Perms.CaseView) ?
							<div className="col-md-6 col-sm-12">
								<VisitsWidget router={this.props.router} />
							</div>
							: null}
						<div className="col-md-6 col-sm-12 form-group">
							<h4>Doctor Onboarding URL <a href="#" onClick={this.onRefreshOnboardURL}><span className="glyphicon glyphicon-refresh"></span></a></h4>
							<input readOnly value={this.state.onboardURL} className="form-control section-name" />
						</div>
					</div>
				</div>
			);
		}
	})
};

var VisitsWidget = React.createClass({displayName: "VisitsWidget",
	mixins: [Routing.RouterNavigateMixin],
	getInitialState: function(): any {
		return {
			error: null,
			busy: true,
			summaries: []
		};
	},
	componentWillMount: function() {
		AdminAPI.visitSummaries("uncompleted", function(success, data, error) {
			if (this.isMounted()) {
				if (!success) {
					this.setState({
						error: error.message,
						busy: false,
					});
					return;
				}
				this.setState({
					summaries: (data.visit_summaries || []).slice(0, 10),
					busy: false,
				});
			}
		}.bind(this));
	},
	render: function(): any {
		var visits = this.state.summaries.map(function(v) {
			var submitted = null;
			if (v.submitted_epoch != 0) {
				var visitSubmitted = new Date(0)
				var currentEpoch = new Date().getTime() / 1000
				visitSubmitted.setUTCSeconds(v.submitted_epoch)
				submitted = "(" + Utils.timeSince(visitSubmitted.getTime()/1000, currentEpoch) + ")";
			}
			return (
				<div key={"visit-" + v.visit_id}>
					<a href={"/case/"+v.case_id+"/visit/"+v.visit_id} onClick={this.onNavigate}>{v.case_name}</a>:
					{" "}{v.status} {submitted}
				</div>
			);
		}.bind(this));
		return (
			<div>
				<h4>Visits</h4>
				{this.state.error ?
					this.state.error
					:
					(this.state.busy ?
						<Utils.LoadingAnimation />
						:
						<div>
							{visits}
						</div>
					)
				}
				<div>
					<a href="/case/visit" onClick={this.onNavigate}>More</a>
				</div>
			</div>
		);
	}
});
