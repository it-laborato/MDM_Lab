import React, { useState, useContext, useCallback, useEffect } from "react";
import { InjectedRouter, Params } from "react-router/lib/Router";
import { useQuery } from "react-query";
import { Tab, Tabs, TabList, TabPanel } from "react-tabs";

import { pick, findIndex } from "lodash";

import { NotificationContext } from "context/notification";
import deviceUserAPI from "services/entities/device_user";
import diskEncryptionAPI from "services/entities/disk_encryption";
import {
  IDeviceMappingResponse,
  IMacadminsResponse,
  IDeviceUserResponse,
  INodeDevice,
} from "interfaces/node";
import { INodePolicy } from "interfaces/policy";
import { IDeviceGlobalConfig } from "interfaces/config";
import { INodeSoftware } from "interfaces/software";

import DeviceUserError from "components/DeviceUserError";
// @ts-ignore
import OrgLogoIcon from "components/icons/OrgLogoIcon";
import Spinner from "components/Spinner";
import Button from "components/buttons/Button";
import TabsWrapper from "components/TabsWrapper";
import Icon from "components/Icon/Icon";
import FlashMessage from "components/FlashMessage";

import { normalizeEmptyValues } from "utilities/helpers";
import PATHS from "router/paths";
import {
  DOCUMENT_TITLE_SUFFIX,
  HOST_ABOUT_DATA,
  HOST_SUMMARY_DATA,
} from "utilities/constants";

import UnsupportedScreenSize from "layouts/UnsupportedScreenSize";

import NodeSummaryCard from "../cards/NodeSummary";
import AboutCard from "../cards/About";
import SoftwareCard from "../cards/Software";
import PoliciesCard from "../cards/Policies";
import InfoModal from "./InfoModal";
import { getErrorMessage } from "./helpers";

import MdmlabIcon from "../../../../../assets/images/mdmlab-avatar-24x24@2x.png";
import PolicyDetailsModal from "../cards/Policies/NodePoliciesTable/PolicyDetailsModal";
import AutoEnrollMdmModal from "./AutoEnrollMdmModal";
import ManualEnrollMdmModal from "./ManualEnrollMdmModal";
import CreateLinuxKeyModal from "./CreateLinuxKeyModal";
import OSSettingsModal from "../OSSettingsModal";
import BootstrapPackageModal from "../NodeDetailsPage/modals/BootstrapPackageModal";
import { parseNodeSoftwareQueryParams } from "../cards/Software/NodeSoftware";
import SelfService from "../cards/Software/SelfService";
import SoftwareDetailsModal from "../cards/Software/SoftwareDetailsModal";
import DeviceUserBanners from "./components/DeviceUserBanners";

const baseClass = "device-user";

const PREMIUM_TAB_PATHS = [
  PATHS.DEVICE_USER_DETAILS,
  PATHS.DEVICE_USER_DETAILS_SELF_SERVICE,
  PATHS.DEVICE_USER_DETAILS_SOFTWARE,
  PATHS.DEVICE_USER_DETAILS_POLICIES,
] as const;

const FREE_TAB_PATHS = [
  PATHS.DEVICE_USER_DETAILS,
  PATHS.DEVICE_USER_DETAILS_SOFTWARE,
] as const;

interface IDeviceUserPageProps {
  location: {
    pathname: string;
    query: {
      vulnerable?: string;
      page?: string;
      query?: string;
      order_key?: string;
      order_direction?: "asc" | "desc";
    };
    search?: string;
  };
  router: InjectedRouter;
  params: Params;
}

const DeviceUserPage = ({
  location,
  router,
  params: { device_auth_token },
}: IDeviceUserPageProps): JSX.Element => {
  const deviceAuthToken = device_auth_token;

  const { renderFlash, notification, hideFlash } = useContext(
    NotificationContext
  );

  const [showInfoModal, setShowInfoModal] = useState(false);
  const [showEnrollMdmModal, setShowEnrollMdmModal] = useState(false);
  const [refetchStartTime, setRefetchStartTime] = useState<number | null>(null);
  const [showRefetchSpinner, setShowRefetchSpinner] = useState(false);
  const [selectedPolicy, setSelectedPolicy] = useState<INodePolicy | null>(
    null
  );
  const [showPolicyDetailsModal, setShowPolicyDetailsModal] = useState(false);
  const [showOSSettingsModal, setShowOSSettingsModal] = useState(false);
  const [showBootstrapPackageModal, setShowBootstrapPackageModal] = useState(
    false
  );
  const [showCreateLinuxKeyModal, setShowCreateLinuxKeyModal] = useState(false);
  const [isTriggeringCreateLinuxKey, setIsTriggeringCreateLinuxKey] = useState(
    false
  );
  const [
    selectedSoftwareDetails,
    setSelectedSoftwareDetails,
  ] = useState<INodeSoftware | null>(null);

  const { data: deviceMapping, refetch: refetchDeviceMapping } = useQuery(
    ["deviceMapping", deviceAuthToken],
    () =>
      deviceUserAPI.loadNodeDetailsExtension(deviceAuthToken, "device_mapping"),
    {
      enabled: !!deviceAuthToken,
      refetchOnMount: false,
      refetchOnReconnect: false,
      refetchOnWindowFocus: false,
      retry: false,
      select: (data: IDeviceMappingResponse) => data.device_mapping,
    }
  );

  const { data: deviceMacAdminsData } = useQuery(
    ["macadmins", deviceAuthToken],
    () => deviceUserAPI.loadNodeDetailsExtension(deviceAuthToken, "macadmins"),
    {
      enabled: !!deviceAuthToken,
      refetchOnMount: false,
      refetchOnReconnect: false,
      refetchOnWindowFocus: false,
      retry: false,
      select: (data: IMacadminsResponse) => data.macadmins,
    }
  );

  const refetchExtensions = () => {
    deviceMapping !== null && refetchDeviceMapping();
  };

  const isRefetching = ({
    refetch_requested,
    refetch_critical_queries_until,
  }: INodeDevice) => {
    if (!refetch_critical_queries_until) {
      return refetch_requested;
    }

    const now = new Date();
    const refetchUntil = new Date(refetch_critical_queries_until);
    const isRefetchingCriticalQueries =
      !isNaN(refetchUntil.getTime()) && refetchUntil > now;
    return refetch_requested || isRefetchingCriticalQueries;
  };

  const {
    data: dupResponse,
    isLoading: isLoadingNode,
    error: loadingDeviceUserError,
    refetch: refetchNodeDetails,
  } = useQuery<IDeviceUserResponse, Error>(
    ["node", deviceAuthToken],
    () =>
      deviceUserAPI.loadNodeDetails({
        token: deviceAuthToken,
        exclude_software: true,
      }),
    {
      enabled: !!deviceAuthToken,
      refetchOnMount: false,
      refetchOnReconnect: false,
      refetchOnWindowFocus: false,
      retry: false,
      onSuccess: ({ node: responseNode }) => {
        setShowRefetchSpinner(isRefetching(responseNode));
        if (isRefetching(responseNode)) {
          // If the API reports that a Mdmlab refetch request is pending, we want to check back for fresh
          // node details. Here we set a one second timeout and poll the API again using
          // fullyReloadNode. We will repeat this process with each onSuccess cycle for a total of
          // 60 seconds or until the API reports that the Mdmlab refetch request has been resolved
          // or that the node has gone offline.
          if (!refetchStartTime) {
            // If our 60 second timer wasn't already started (e.g., if a refetch was pending when
            // the first page loads), we start it now if the node is online. If the node is offline,
            // we skip the refetch on page load.
            if (responseNode.status === "online") {
              setRefetchStartTime(Date.now());
              setTimeout(() => {
                refetchNodeDetails();
                refetchExtensions();
              }, 1000);
            } else {
              setShowRefetchSpinner(false);
            }
          } else {
            const totalElapsedTime = Date.now() - refetchStartTime;
            if (totalElapsedTime < 60000) {
              if (responseNode.status === "online") {
                setTimeout(() => {
                  refetchNodeDetails();
                  refetchExtensions();
                }, 1000);
              } else {
                renderFlash(
                  "error",
                  `This node is offline. Please try refetching node vitals later.`
                );
                setShowRefetchSpinner(false);
              }
            } else {
              renderFlash(
                "error",
                `We're having trouble fetching fresh vitals for this node. Please try again later.`
              );
              setShowRefetchSpinner(false);
            }
          }
          // exit early because refectch is pending so we can avoid unecessary steps below
        }
      },
    }
  );

  const {
    node,
    license,
    org_logo_url: orgLogoURL = "",
    org_contact_url: orgContactURL = "",
    global_config: globalConfig = null as IDeviceGlobalConfig | null,
    self_service: hasSelfService = false,
  } = dupResponse || {};
  const isPremiumTier = license?.tier === "premium";

  const summaryData = normalizeEmptyValues(pick(node, HOST_SUMMARY_DATA));

  const aboutData = normalizeEmptyValues(pick(node, HOST_ABOUT_DATA));

  const toggleInfoModal = useCallback(() => {
    setShowInfoModal(!showInfoModal);
  }, [showInfoModal, setShowInfoModal]);

  const toggleEnrollMdmModal = useCallback(() => {
    setShowEnrollMdmModal(!showEnrollMdmModal);
  }, [showEnrollMdmModal, setShowEnrollMdmModal]);

  const togglePolicyDetailsModal = useCallback(
    (policy: INodePolicy) => {
      setShowPolicyDetailsModal(!showPolicyDetailsModal);
      setSelectedPolicy(policy);
    },
    [showPolicyDetailsModal, setShowPolicyDetailsModal, setSelectedPolicy]
  );

  const bootstrapPackageData = {
    status: node?.mdm.macos_setup?.bootstrap_package_status,
    details: node?.mdm.macos_setup?.details,
    name: node?.mdm.macos_setup?.bootstrap_package_name,
  };

  const toggleOSSettingsModal = useCallback(() => {
    setShowOSSettingsModal(!showOSSettingsModal);
  }, [showOSSettingsModal, setShowOSSettingsModal]);

  const onCancelPolicyDetailsModal = useCallback(() => {
    setShowPolicyDetailsModal(!showPolicyDetailsModal);
    setSelectedPolicy(null);
  }, [showPolicyDetailsModal, setShowPolicyDetailsModal, setSelectedPolicy]);

  const onRefetchNode = async () => {
    if (node) {
      setShowRefetchSpinner(true);
      try {
        await deviceUserAPI.refetch(deviceAuthToken);
        setRefetchStartTime(Date.now());
        setTimeout(() => {
          refetchNodeDetails();
          refetchExtensions();
        }, 1000);
      } catch (error) {
        renderFlash("error", getErrorMessage(error, node.display_name));
        setShowRefetchSpinner(false);
      }
    }
  };

  // Updates title that shows up on browser tabs
  useEffect(() => {
    document.title = `My device | ${DOCUMENT_TITLE_SUFFIX}`;
  }, [location.pathname, node]);

  const renderActionButtons = () => {
    return (
      <div className={`${baseClass}__action-button-container`}>
        <Button onClick={() => setShowInfoModal(true)} variant="text-icon">
          <>
            Info <Icon name="info" size="small" />
          </>
        </Button>
      </div>
    );
  };

  const renderEnrollMdmModal = () => {
    return node?.dep_assigned_to_mdmlab ? (
      <AutoEnrollMdmModal node={node} onCancel={toggleEnrollMdmModal} />
    ) : (
      <ManualEnrollMdmModal
        onCancel={toggleEnrollMdmModal}
        token={deviceAuthToken}
      />
    );
  };

  const onTriggerEscrowLinuxKey = async () => {
    setIsTriggeringCreateLinuxKey(true);
    // modal opens in loading state
    setShowCreateLinuxKeyModal(true);
    try {
      await diskEncryptionAPI.triggerLinuxDiskEncryptionKeyEscrow(
        deviceAuthToken
      );
    } catch (e) {
      renderFlash("error", "Failed to trigger key creation.");
      setShowCreateLinuxKeyModal(false);
    } finally {
      setIsTriggeringCreateLinuxKey(false);
    }
  };

  const renderDeviceUserPage = () => {
    const failingPoliciesCount = node?.issues?.failing_policies_count || 0;

    // TODO: We should probably have a standard way to handle this on all pages. Do we want to show
    // a premium-only message in the case that a user tries direct navigation to a premium-only page
    // or silently redirect as below?
    let tabPaths = (isPremiumTier
      ? PREMIUM_TAB_PATHS
      : FREE_TAB_PATHS
    ).map((t) => t(deviceAuthToken));
    if (!hasSelfService) {
      tabPaths = tabPaths.filter((path) => !path.includes("self-service"));
    }

    const findSelectedTab = (pathname: string) =>
      findIndex(tabPaths, (x) => x.startsWith(pathname.split("?")[0]));
    if (!isLoadingNode && node && findSelectedTab(location.pathname) === -1) {
      router.push(tabPaths[0]);
    }

    // Note: API response global_config is misnamed because the backend actually returns the global
    // or team config (as applicable)
    const isSoftwareEnabled = !!globalConfig?.features
      ?.enable_software_inventory;

    return (
      <div className="core-wrapper">
        {!node || isLoadingNode ? (
          <Spinner />
        ) : (
          <div className={`${baseClass} main-content`}>
            <DeviceUserBanners
              nodePlatform={node.platform}
              nodeOsVersion={node.os_version}
              mdmEnrollmentStatus={node.mdm.enrollment_status}
              mdmEnabledAndConfigured={
                !!globalConfig?.mdm.enabled_and_configured
              }
              connectedToMdmlabMdm={!!node.mdm.connected_to_mdmlab}
              macDiskEncryptionStatus={
                node.mdm.macos_settings?.disk_encryption ?? null
              }
              diskEncryptionActionRequired={
                node.mdm.macos_settings?.action_required ?? null
              }
              onTurnOnMdm={toggleEnrollMdmModal}
              onTriggerEscrowLinuxKey={onTriggerEscrowLinuxKey}
              diskEncryptionOSSetting={node.mdm.os_settings?.disk_encryption}
              diskIsEncrypted={node.disk_encryption_enabled}
              diskEncryptionKeyAvailable={node.mdm.encryption_key_available}
            />
            <NodeSummaryCard
              summaryData={summaryData}
              bootstrapPackageData={bootstrapPackageData}
              isPremiumTier={isPremiumTier}
              toggleOSSettingsModal={toggleOSSettingsModal}
              nodeSettings={node?.mdm.profiles ?? []}
              showRefetchSpinner={showRefetchSpinner}
              onRefetchNode={onRefetchNode}
              renderActionDropdown={renderActionButtons}
              osSettings={node?.mdm.os_settings}
              deviceUser
            />
            <TabsWrapper className={`${baseClass}__tabs-wrapper`}>
              <Tabs
                selectedIndex={findSelectedTab(location.pathname)}
                onSelect={(i) => router.push(tabPaths[i])}
              >
                <TabList>
                  <Tab>Details</Tab>
                  {isPremiumTier && isSoftwareEnabled && hasSelfService && (
                    <Tab>Self-service</Tab>
                  )}
                  {isSoftwareEnabled && <Tab>Software</Tab>}
                  {isPremiumTier && (
                    <Tab>
                      <div>
                        {failingPoliciesCount > 0 && (
                          <span className="count">{failingPoliciesCount}</span>
                        )}
                        Policies
                      </div>
                    </Tab>
                  )}
                </TabList>
                <TabPanel>
                  <AboutCard
                    aboutData={aboutData}
                    deviceMapping={deviceMapping}
                    munki={deviceMacAdminsData?.munki}
                  />
                </TabPanel>
                {isPremiumTier && isSoftwareEnabled && hasSelfService && (
                  <TabPanel>
                    <SelfService
                      contactUrl={orgContactURL}
                      deviceToken={deviceAuthToken}
                      isSoftwareEnabled
                      pathname={location.pathname}
                      queryParams={parseNodeSoftwareQueryParams(location.query)}
                      router={router}
                    />
                  </TabPanel>
                )}
                {isSoftwareEnabled && (
                  <TabPanel>
                    <SoftwareCard
                      id={deviceAuthToken}
                      softwareUpdatedAt={node.software_updated_at}
                      nodeCanWriteSoftware={!!node.orbit_version}
                      router={router}
                      pathname={location.pathname}
                      queryParams={parseNodeSoftwareQueryParams(location.query)}
                      isMyDevicePage
                      platform={node.platform}
                      nodeTeamId={node.team_id || 0}
                      isSoftwareEnabled={isSoftwareEnabled}
                      onShowSoftwareDetails={setSelectedSoftwareDetails}
                    />
                  </TabPanel>
                )}
                {isPremiumTier && (
                  <TabPanel>
                    <PoliciesCard
                      policies={node?.policies || []}
                      isLoading={isLoadingNode}
                      deviceUser
                      togglePolicyDetailsModal={togglePolicyDetailsModal}
                      nodePlatform={node?.platform || ""}
                      router={router}
                    />
                  </TabPanel>
                )}
              </Tabs>
            </TabsWrapper>
            {showInfoModal && <InfoModal onCancel={toggleInfoModal} />}
            {showEnrollMdmModal && renderEnrollMdmModal()}
          </div>
        )}
        {!!node && showPolicyDetailsModal && (
          <PolicyDetailsModal
            onCancel={onCancelPolicyDetailsModal}
            policy={selectedPolicy}
          />
        )}
        {!!node && showOSSettingsModal && (
          <OSSettingsModal
            canResendProfiles={false}
            nodeId={node.id}
            platform={node.platform}
            nodeMDMData={node.mdm}
            onClose={toggleOSSettingsModal}
          />
        )}
        {showBootstrapPackageModal &&
          bootstrapPackageData.details &&
          bootstrapPackageData.name && (
            <BootstrapPackageModal
              packageName={bootstrapPackageData.name}
              details={bootstrapPackageData.details}
              onClose={() => setShowBootstrapPackageModal(false)}
            />
          )}
        {showCreateLinuxKeyModal && !!node && (
          <CreateLinuxKeyModal
            isTriggeringCreateLinuxKey={isTriggeringCreateLinuxKey}
            onExit={() => {
              setShowCreateLinuxKeyModal(false);
            }}
          />
        )}
        {selectedSoftwareDetails && !!node && (
          <SoftwareDetailsModal
            nodeDisplayName={node.display_name}
            software={selectedSoftwareDetails}
            onExit={() => setSelectedSoftwareDetails(null)}
            hideInstallDetails
          />
        )}
      </div>
    );
  };

  return (
    <div className="app-wrap">
      <UnsupportedScreenSize />
      <FlashMessage
        fullWidth
        notification={notification}
        onRemoveFlash={hideFlash}
        pathname={location.pathname}
      />
      <nav className="site-nav-container">
        <div className="site-nav-content">
          <ul className="site-nav-list">
            <li className="site-nav-item dup-org-logo" key="dup-org-logo">
              <div className="site-nav-item__logo-wrapper">
                <div className="site-nav-item__logo">
                  <OrgLogoIcon className="logo" src={orgLogoURL || MdmlabIcon} />
                </div>
              </div>
            </li>
          </ul>
        </div>
      </nav>
      {loadingDeviceUserError ? <DeviceUserError /> : renderDeviceUserPage()}
    </div>
  );
};

export default DeviceUserPage;
