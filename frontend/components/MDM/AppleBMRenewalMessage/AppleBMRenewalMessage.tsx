import React from "react";

import InfoBanner from "components/InfoBanner";

const baseClass = "apple-bm-renewal-message";

type IAppleBMRenewalMessageProps = {
  expired: boolean;
};

const AppleBMRenewalMessage = ({ expired }: IAppleBMRenewalMessageProps) => {
  return (
    <InfoBanner
      className={baseClass}
      color="yellow"
     
    >
      {expired ? (
        <>
          Your Apple Business Manager (ABM) server token has expired. macOS,
          iOS, and iPadOS nodes won’t automatically enroll to Mdmlab. Users with
          the admin role in Mdmlab can renew ABM.
        </>
      ) : (
        <>
          Your Apple Business Manager (ABM) server token is less than 30 days
          from expiration. If it expires, macOS, iOS, and iPadOS nodes won’t
          automatically enroll to Mdmlab. Users with the admin role in Mdmlab can
          renew ABM.
        </>
      )}
    </InfoBanner>
  );
};

export default AppleBMRenewalMessage;
