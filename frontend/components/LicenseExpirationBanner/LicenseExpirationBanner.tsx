import React from "react";

import InfoBanner from "components/InfoBanner";

const baseClass = "license-expiry-banner";

const LicenseExpirationBanner = (): JSX.Element => {
  return (
    <InfoBanner
      className={baseClass}
      color="yellow"
      
    >
      Your Mdmlab Premium license is about to expire.
    </InfoBanner>
  );
};

export default LicenseExpirationBanner;
