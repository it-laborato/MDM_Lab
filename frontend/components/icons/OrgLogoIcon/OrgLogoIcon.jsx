import React, { Component } from "react";
import PropTypes from "prop-types";
import classnames from "classnames";

import mdmlabAvatar from "../../../../assets/images/mdmlab-avatar-24x24@2x.png";

const baseClass = "org-logo-icon";

class OrgLogoIcon extends Component {
  static propTypes = {
    className: PropTypes.string,
    src: PropTypes.string.isRequired,
  };

  static defaultProps = {
    src: mdmlabAvatar,
  };

  constructor(props) {
    super(props);

    this.state = { imageSrc: mdmlabAvatar };
  }

  componentWillMount() {
    const { src } = this.props;

    this.setState({ imageSrc: src });

    return false;
  }

  componentWillReceiveProps(nextProps) {
    const { src } = nextProps;
    const { unchangedSourceProp } = this;

    if (unchangedSourceProp(nextProps)) {
      return false;
    }

    this.setState({ imageSrc: src });

    return false;
  }

  shouldComponentUpdate(nextProps) {
    const { imageSrc } = this.state;
    const { unchangedSourceProp } = this;

    if (unchangedSourceProp(nextProps) && imageSrc === mdmlabAvatar) {
      return false;
    }

    return true;
  }

  onError = () => {
    this.setState({ imageSrc: mdmlabAvatar });

    return false;
  };

  unchangedSourceProp = (nextProps) => {
    const { src: nextSrcProp } = nextProps;
    const { src } = this.props;

    return src === nextSrcProp;
  };

  render() {
    const { className } = this.props;
    const { imageSrc } = this.state;
    const { onError } = this;

    const classNames =
      imageSrc === mdmlabAvatar
        ? classnames(baseClass, className, "default-mdmlab-logo")
        : classnames(baseClass, className);

    return (
      <img
        alt="Organization Logo"
        className={classNames}
        onError={onError}
        src={imageSrc}
      />
    );
  }
}

export default OrgLogoIcon;
