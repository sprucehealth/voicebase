/* @flow */

var React = require("react");
var routing = require("./routing.js");

var TopNav = React.createClass({displayName: "TopNav",
	mixins: [routing.RouterNavigateMixin],
	propTypes: {
		name: React.PropTypes.string.isRequired,
		activeItem: React.PropTypes.string,
		leftItems: React.PropTypes.arrayOf(React.PropTypes.shape({
			id: React.PropTypes.string.isRequired,
			name: React.PropTypes.node.isRequired,
			url: React.PropTypes.string.isRequired
		})).isRequired,
		rightItems: React.PropTypes.arrayOf(React.PropTypes.shape({
			id: React.PropTypes.string.isRequired,
			name: React.PropTypes.node.isRequired,
			url: React.PropTypes.string.isRequired
		})).isRequired,
		router: React.PropTypes.object.isRequired
	},
	render: function(): any {
		var leftMenuItems = this.props.leftItems.map(function(item) {
			var active = item.id == this.props.activeItem;
			return (
				<li key={item.id} className={active ? 'active' : ''}><a href={this.props.router.root + item.url} onClick={this.onNavigate}>{item.name}</a></li>
			);
		}.bind(this));
		var rightMenuItems = this.props.rightItems.map(function(item) {
			var active = item.id == this.props.activeItem;
			return (
				<li key={item.id} className={active ? 'active' : ''}><a href={this.props.router.root + item.url} onClick={this.onNavigate}>{item.name}</a></li>
			);
		}.bind(this));
		return (
			<div className="navbar navbar-inverse navbar-fixed-top" role="navigation">
				<div className="container-fluid">
					<div className="navbar-header">
						<button type="button" className="navbar-toggle" data-toggle="collapse" data-target=".navbar-collapse">
							<span className="sr-only">Toggle navigation</span>
							<span className="icon-bar"></span>
							<span className="icon-bar"></span>
							<span className="icon-bar"></span>
						</button>
						<a className="navbar-brand" href={this.props.router.root} onClick={this.onNavigate}>{this.props.name}</a>
					</div>
					<div className="collapse navbar-collapse">
						<ul className="nav navbar-nav">
							{leftMenuItems}
						</ul>
						<ul className="nav navbar-nav navbar-right">
							{rightMenuItems}
							<li><a href="/logout">Sign Out</a></li>
						</ul>
					</div>
				</div>
			</div>
		);
	}
});

var LeftNav = React.createClass({displayName: "LeftNav",
	mixins: [routing.RouterNavigateMixin],
	propTypes: {
		currentPage: React.PropTypes.string,
		items: React.PropTypes.arrayOf(
			React.PropTypes.arrayOf(
				React.PropTypes.shape({
				id: React.PropTypes.oneOfType([
					React.PropTypes.string,
					React.PropTypes.number,
				]).isRequired,
				name: React.PropTypes.node.isRequired,
				url: React.PropTypes.string.isRequired
			}))).isRequired,
		router: React.PropTypes.object.isRequired
	},
	render: function(): any {
		var navGroups = this.props.items.map(function(subItems, index) {
			return (
				<LeftNavItemGroup key={"leftNavGroup-"+index}>
					{subItems.map(function(item) {
						var active = item.active || (item.id == this.props.currentPage);
						return <LeftNavItem router={this.props.router} key={item.id} active={active} url={item.url} heading={item.heading} name={item.name} />;
					}.bind(this))}
				</LeftNavItemGroup>
			);
		}.bind(this));
		return (
			<div>
				<div className="row">
					<div className="col-sm-3 col-md-2 sidebar">
						{navGroups}
					</div>
				</div>
				<div className="col-sm-9 col-sm-offset-3 col-md-10 col-md-offset-2 main">
					{this.props.children}
				</div>
			</div>
		);
	}
});

var LeftNavItemGroup = React.createClass({displayName: "LeftNavItemGroup",
	render: function(): any {
		return (
			<ul className="nav nav-sidebar">
				{this.props.children}
			</ul>
		);
	}
});

var LeftNavItem = React.createClass({displayName: "LeftNavItem",
	mixins: [routing.RouterNavigateMixin],
	render: function(): any {
		var click = this.props.onClick || this.onNavigate;
		return (
			<li className={this.props.active?"active":""}>
				<a href={this.props.url} onClick={click} className={this.props.heading?"heading":""}>{this.props.name}</a>
			</li>
		);
	}
});

module.exports = {
	TopNav: TopNav,
	LeftNav: LeftNav
};
