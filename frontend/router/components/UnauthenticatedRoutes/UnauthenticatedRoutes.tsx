import React from "react";

interface IAppProps {
  children: JSX.Element;
}

export const UnauthenticatedRoutes = ({ children }: IAppProps): JSX.Element => {
  if (window.location.hostname.includes(".sandbox.mdmlabdm.com")) {
    window.location.href = "https://www.mdmlabdm.com/try-mdmlab/login";
  }
  return <div>{children}</div>;
};

export default UnauthenticatedRoutes;
