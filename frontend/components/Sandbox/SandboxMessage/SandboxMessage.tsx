import classnames from "classnames";
import React from "react";


interface ISandboxMessageProps {
  variant?: "demo" | "sales";
  /** message to display in the sandbox error */
  message: string;
  /** UTM (Urchin Tracking Module) source text that is added to the demo link */
  utmSource?: string;
  className?: string;
}

const baseClass = "sandbox-message";

const SandboxMessage = ({
  variant = "demo",
  message,
  utmSource,
  className,
}: ISandboxMessageProps): JSX.Element => {
  const classes = classnames(baseClass, className);
 


  return (
    <div className={classes}>
      <h2 className={`${baseClass}__message`}>{message}</h2>
      
    </div>
  );
};

export default SandboxMessage;
