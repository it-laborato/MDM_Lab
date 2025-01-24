import React from "react";

import Icon from "components/Icon";
import classnames from "classnames";
import { Colors } from "styles/var/colors";

interface ICustomLinkProps {
  url: string;
  text: string;
  className?: string;
  newTab?: boolean;
  /** Icon wraps on new line with last word */
  multiline?: boolean;
  iconColor?: Colors;
  color?: "core-mdmlab-blue" | "core-mdmlab-black" | "core-mdmlab-white";
  /** Restricts access via keyboard when CustomLink is part of disabled UI */
  disableKeyboardNavigation?: boolean;
}

const baseClass = "custom-link";

const CustomLink = ({
  url,
  text,
  className,
  newTab = false,
  multiline = false,
  iconColor = "core-mdmlab-blue",
  color = "core-mdmlab-blue",
  disableKeyboardNavigation = false,
}: ICustomLinkProps): JSX.Element => {
  const customLinkClass = classnames(baseClass, className, {
    [`${baseClass}--black`]: color === "core-mdmlab-black",
    [`${baseClass}--white`]: color === "core-mdmlab-white",
  });

  const target = newTab ? "_blank" : "";

  const multilineText = text.substring(0, text.lastIndexOf(" ") + 1);
  const lastWord = text.substring(text.lastIndexOf(" ") + 1, text.length);

  const content = multiline ? (
    <>
      {multilineText}
      <span className={`${baseClass}__no-wrap`}>
        {lastWord}
        {newTab && (
          <Icon
            name="external-link"
            className={`${baseClass}__external-icon`}
            color={iconColor}
          />
        )}
      </span>
    </>
  ) : (
    <>
      {text}
      {newTab && (
        <Icon
          name="external-link"
          className={`${baseClass}__external-icon`}
          color={iconColor}
        />
      )}
    </>
  );

  return (
    <a
      href={url}
      target={target}
      rel="noopener noreferrer"
      className={customLinkClass}
      tabIndex={disableKeyboardNavigation ? -1 : 0}
    >
      {content}
    </a>
  );
};

export default CustomLink;
