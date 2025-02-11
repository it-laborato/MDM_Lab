import React from "react";

import InfoBanner from "components/InfoBanner";

const baseClass = "apple-pn-cert-renewal-message";

type IApplePNCertRenewalMessage = {
  expired: boolean;
};

const ApplePNCertRenewalMessage = ({ expired }: IApplePNCertRenewalMessage) => {
  return (
    <InfoBanner
      className={baseClass}
      color="yellow"
     
    >
      {expired ? (
        <>
          Your Apple Push Notification service (APNs) certificate has expired.
          After you renew the certificate, all end users have to turn MDM off
          and back on.
        </>
      ) : (
        <>
          Your Apple Push Notification service (APNs) certificate is less than
          30 days from expiration. If it expires all end users will have to turn
          MDM off and back on.
        </>
      )}
    </InfoBanner>
  );
};

export default ApplePNCertRenewalMessage;
