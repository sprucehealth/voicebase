/** @jsx React.DOM */

var AdminAPI = require("./api.js");
var Forms = require("../forms.js");
var Routing = require("../routing.js");
var Utils = require("../utils.js");

module.exports = {
	Menu: React.createClass({displayName: "PathwaysMenu",
		mixins: [Routing.RouterNavigateMixin],
		getInitialState: function() {
			return {
				menu_json: null,
				busy: false,
				error: null
			};
		},
		componentWillMount: function() {
			document.title = "Pathways | Menu";
			this.setState({busy: true});
			AdminAPI.pathwayMenu(function(success, data, error) {
				if (this.isMounted()) {
					if (success) {
						this.setState({
							busy: false,
							error: null,
							menu_json: JSON.stringify(data, null, 4)
						});
					} else {
						this.setState({busy: false, error: error.message});
					}
				}
			}.bind(this));
		},
		onChange: function(e) {
			e.preventDefault();
			var error = null;
			try {
				JSON.parse(e.target.value)
			} catch(ex) {
				error = "Invalid JSON: " + ex.message;
			}
			this.setState({
				error: error,
				menu_json: e.target.value
			});
		},
		onSubmit: function(e) {
			e.preventDefault();
			try {
				var menu = JSON.parse(this.state.menu_json);
			} catch(ex) {
				this.setState({error: "Invalid JSON: " + ex.message});
				return;
			}
			this.setState({busy: true});
			AdminAPI.updatePathwayMenu(menu, function(success, data, error) {
				if (this.isMounted()) {
					if (success) {
						this.setState({
							busy: false,
							error: null,
							menu_json: JSON.stringify(data, null, 4)
						});
					} else {
						this.setState({busy: false, error: error.message});
					}
				}
			}.bind(this));
		},
		render: function() {
			return (
				<div className="container">
					<div className="row">
						<div className="col-sm-12 col-md-12">
							<h2>Pathways Menu</h2>
							{this.state.menu_json ?
								<form role="form" onSubmit={this.onSubmit} method="PUT">
									<div>
										<Forms.TextArea name="json" required label="JSON" value={this.state.menu_json} rows="20" onChange={this.onChange} />
									</div>
									<div className="text-right">
										{this.state.error ? <Utils.Alert type="danger">{this.state.error}</Utils.Alert> : null}
										{this.state.busy ? <Utils.LoadingAnimation /> : null}
										<button type="submit" className="btn btn-primary" disabled={this.state.error || this.state.error ? true : false}>Save</button>
									</div>
								</form>
								:
								null
							}
						</div>
					</div>
				</div>
			);
		}
	})
};
