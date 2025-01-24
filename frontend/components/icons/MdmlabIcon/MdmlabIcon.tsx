import React from "react";
import classnames from "classnames";

interface IMdmlabIconProps {
  className?: string;
  fw?: boolean;
  name: string;
  size?: string;
  title?: string;
}

const baseClass = "fleeticon";

const MdmlabIcon = ({
  className,
  fw,
  name,
  size,
  title,
}: IMdmlabIconProps): JSX.Element => {
  const iconClasses = classnames(baseClass, `${baseClass}-${name}`, className, {
    [`${baseClass}-fw`]: fw,
    [`${baseClass}-${size}`]: !!size,
  });

  return <i className={iconClasses} title={title} />;
};

export default MdmlabIcon;
