import React from "react";

import InfoBanner from "components/InfoBanner";

const baseClass = "apple-bm-terms-message";

const AppleBMTermsMessage = () => {
  return (
    <InfoBanner
      className={baseClass}
      color="yellow"
      
    >
      You can’t automatically enroll macOS, iOS, and iPadOS nodes until you
      accept the new terms and conditions for your Apple Business Manager (ABM).
      An ABM administrator can accept these terms.
    </InfoBanner>
  );
};

export default AppleBMTermsMessage;
