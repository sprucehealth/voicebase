/** @jsx React.DOM */

var AdminAPI = require("./api.js");
var Routing = require("../routing.js");

module.exports = {
	Dashboard: React.createClass({displayName: "Dashboard",
		mixins: [Routing.RouterNavigateMixin],
		getInitialState: function() {
			return {onboardURL: ""}
		},
		componentWillMount: function() {
			document.title = "Dashboard | Spruce Admin";
			this.onRefreshOnboardURL();
		},
		onRefreshOnboardURL: function() {
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
	})
};
