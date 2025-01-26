import React, {
  useState,
  useContext,
  useEffect,
  useCallback,
  useMemo,
} from "react";
import { useQuery } from "react-query";
import { InjectedRouter, Params } from "react-router/lib/Router";
import { RouteProps } from "react-router/lib/Route";
import { find, isEmpty, isEqual, omit } from "lodash";
import { format } from "date-fns";
import FileSaver from "file-saver";

import enrollSecretsAPI from "services/entities/enroll_secret";
import usersAPI from "services/entities/users";
import labelsAPI, { ILabelsResponse } from "services/entities/labels";
import teamsAPI, { ILoadTeamsResponse } from "services/entities/teams";
import globalPoliciesAPI from "services/entities/global_policies";
import nodesAPI, {
  HOSTS_QUERY_PARAMS as PARAMS,
  ILoadNodesQueryKey,
  ILoadNodesResponse,
  ISortOption,
  MacSettingsStatusQueryParam,
  HOSTS_QUERY_PARAMS,
} from "services/entities/nodes";
import nodeCountAPI, {
  INodesCountQueryKey,
  INodesCountResponse,
} from "services/entities/node_count";
import {
  getOSVersions,
  IGetOSVersionsQueryKey,
  IOSVersionsResponse,
} from "services/entities/operating_systems";

import PATHS from "router/paths";
import { AppContext } from "context/app";
import { TableContext } from "context/table";
import { NotificationContext } from "context/notification";

import useTeamIdParam from "hooks/useTeamIdParam";

import {
  IEnrollSecret,
  IEnrollSecretsResponse,
} from "interfaces/enroll_secret";
import { ILabel } from "interfaces/label";
import { IOperatingSystemVersion } from "interfaces/operating_system";
import { IPolicy, IStoredPolicyResponse } from "interfaces/policy";
import {
  isValidSoftwareAggregateStatus,
  SoftwareAggregateStatus,
} from "interfaces/software";
import { API_ALL_TEAMS_ID, ITeam } from "interfaces/team";
import { IEmptyTableProps } from "interfaces/empty_table";
import {
  DiskEncryptionStatus,
  BootstrapPackageStatus,
  MdmProfileStatus,
} from "interfaces/mdm";

import sortUtils from "utilities/sort";
import {
  HOSTS_SEARCH_BOX_PLACEHOLDER,
  HOSTS_SEARCH_BOX_TOOLTIP,
  PolicyResponse,
} from "utilities/constants";
import { getNextLocationPath } from "utilities/helpers";

import Button from "components/buttons/Button";
import Icon from "components/Icon/Icon";
import { SingleValue } from "react-select-5";
import DropdownWrapper from "components/forms/fields/DropdownWrapper";
import { CustomOptionType } from "components/forms/fields/DropdownWrapper/DropdownWrapper";
import TableContainer from "components/TableContainer";
import InfoBanner from "components/InfoBanner/InfoBanner";
import { ITableQueryData } from "components/TableContainer/TableContainer";
import TableCount from "components/TableContainer/TableCount";
import TableDataError from "components/DataError";
import { IActionButtonProps } from "components/TableContainer/DataTable/ActionButton/ActionButton";
import TeamsDropdown from "components/TeamsDropdown";
import Spinner from "components/Spinner";
import MainContent from "components/MainContent";
import EmptyTable from "components/EmptyTable";
import {
  defaultHiddenColumns,
  generateVisibleTableColumns,
  generateAvailableTableHeaders,
} from "./NodeTableConfig";
import {
  LABEL_SLUG_PREFIX,
  DEFAULT_SORT_HEADER,
  DEFAULT_SORT_DIRECTION,
  DEFAULT_PAGE_SIZE,
  DEFAULT_PAGE_INDEX,
  nodeSelectStatuses,
  MANAGE_HOSTS_PAGE_FILTER_KEYS,
  MANAGE_HOSTS_PAGE_LABEL_INCOMPATIBLE_QUERY_PARAMS,
} from "./NodesPageConfig";
import { getDeleteLabelErrorMessages, isAcceptableStatus } from "./helpers";

import DeleteSecretModal from "../../../components/EnrollSecrets/DeleteSecretModal";
import SecretEditorModal from "../../../components/EnrollSecrets/SecretEditorModal";
import AddNodesModal from "../../../components/AddNodesModal";
import EnrollSecretModal from "../../../components/EnrollSecrets/EnrollSecretModal";
// @ts-ignore
import EditColumnsModal from "./components/EditColumnsModal/EditColumnsModal";
import TransferNodeModal from "../components/TransferNodeModal";
import DeleteNodeModal from "../components/DeleteNodeModal";
import DeleteLabelModal from "./components/DeleteLabelModal";
import LabelFilterSelect from "./components/LabelFilterSelect";
import NodesFilterBlock from "./components/NodesFilterBlock";

interface IManageNodesProps {
  route: RouteProps;
  router: InjectedRouter;
  params: Params;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  location: any; // no type in react-router v3 TODO: Improve this type
}

const CSV_HOSTS_TITLE = "Nodes";
const baseClass = "manage-nodes";

const ManageNodesPage = ({
  route,
  router,
  params: routeParams,
  location,
}: IManageNodesProps): JSX.Element => {
  const routeTemplate = route?.path ?? "";
  const queryParams = location.query;
  const {
    config,
    currentUser,
    filteredNodesPath,
    isGlobalAdmin,
    isGlobalMaintainer,
    isOnGlobalTeam,
    isOnlyObserver,
    isPremiumTier,
    isFreeTier,
    isSandboxMode,
    userSettings,
    setFilteredNodesPath,
    setFilteredPoliciesPath,
    setFilteredQueriesPath,
    setFilteredSoftwarePath,
    setUserSettings,
  } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  const { setResetSelectedRows } = useContext(TableContext);

  const {
    currentTeamId,
    currentTeamName,
    isAnyTeamSelected,
    isRouteOk,
    isTeamAdmin,
    isTeamMaintainer,
    isTeamMaintainerOrTeamAdmin,
    teamIdForApi,
    userTeams,
    handleTeamChange,
  } = useTeamIdParam({
    location,
    router,
    includeAllTeams: true,
    includeNoTeam: true,
    overrideParamsOnTeamChange: {
      // remove the software status filter when selecting All teams
      [HOSTS_QUERY_PARAMS.SOFTWARE_STATUS]: (newTeamId?: number) =>
        newTeamId === API_ALL_TEAMS_ID,
    },
  });

  // Functions to avoid race conditions
  const initialSortBy: ISortOption[] = (() => {
    let key = DEFAULT_SORT_HEADER;
    let direction = DEFAULT_SORT_DIRECTION;

    if (queryParams) {
      const { order_key, order_direction } = queryParams;
      key = order_key || key;
      direction = order_direction || direction;
    }

    return [{ key, direction }];
  })();
  const initialQuery = (() => queryParams.query ?? "")();
  const initialPage = (() =>
    queryParams && queryParams.page ? parseInt(queryParams?.page, 10) : 0)();

  // ========= states
  const [selectedLabel, setSelectedLabel] = useState<ILabel>();
  const [selectedSecret, setSelectedSecret] = useState<IEnrollSecret>();
  const [showNoEnrollSecretBanner, setShowNoEnrollSecretBanner] = useState(
    true
  );
  const [showDeleteSecretModal, setShowDeleteSecretModal] = useState(false);
  const [showSecretEditorModal, setShowSecretEditorModal] = useState(false);
  const [showEnrollSecretModal, setShowEnrollSecretModal] = useState(false);
  const [showDeleteLabelModal, setShowDeleteLabelModal] = useState(false);
  const [showEditColumnsModal, setShowEditColumnsModal] = useState(false);
  const [showAddNodesModal, setShowAddNodesModal] = useState(false);
  const [showTransferNodeModal, setShowTransferNodeModal] = useState(false);
  const [showDeleteNodeModal, setShowDeleteNodeModal] = useState(false);
  const [hiddenColumns, setHiddenColumns] = useState<string[]>(
    userSettings?.hidden_node_columns || defaultHiddenColumns
  );
  const [selectedNodeIds, setSelectedNodeIds] = useState<number[]>([]);
  const [isAllMatchingNodesSelected, setIsAllMatchingNodesSelected] = useState(
    false
  );
  const [searchQuery, setSearchQuery] = useState(initialQuery);
  const [page, setPage] = useState(initialPage);
  const [sortBy, setSortBy] = useState<ISortOption[]>(initialSortBy);
  const [tableQueryData, setTableQueryData] = useState<ITableQueryData>();
  const [resetPageIndex, setResetPageIndex] = useState<boolean>(false);
  const [isUpdatingLabel, setIsUpdatingLabel] = useState<boolean>(false);
  const [isUpdatingSecret, setIsUpdatingSecret] = useState<boolean>(false);
  const [isUpdatingNodes, setIsUpdatingNodes] = useState<boolean>(false);

  // ========= queryParams
  const policyId = queryParams?.policy_id;
  const policyResponse: PolicyResponse = queryParams?.policy_response;
  const macSettingsStatus = queryParams?.macos_settings;
  const softwareId =
    queryParams?.software_id !== undefined
      ? parseInt(queryParams.software_id, 10)
      : undefined;
  const softwareVersionId =
    queryParams?.software_version_id !== undefined
      ? parseInt(queryParams.software_version_id, 10)
      : undefined;
  const softwareTitleId =
    queryParams?.software_title_id !== undefined
      ? parseInt(queryParams.software_title_id, 10)
      : undefined;
  const softwareStatus = isValidSoftwareAggregateStatus(
    queryParams?.[HOSTS_QUERY_PARAMS.SOFTWARE_STATUS]
  )
    ? (queryParams[
        HOSTS_QUERY_PARAMS.SOFTWARE_STATUS
      ] as SoftwareAggregateStatus)
    : undefined;
  const status = isAcceptableStatus(queryParams?.status)
    ? queryParams?.status
    : undefined;
  const mdmId =
    queryParams?.mdm_id !== undefined
      ? parseInt(queryParams.mdm_id, 10)
      : undefined;
  const mdmEnrollmentStatus = queryParams?.mdm_enrollment_status;
  const {
    os_version_id: osVersionId,
    os_name: osName,
    os_version: osVersion,
  } = queryParams;
  const vulnerability = queryParams?.vulnerability;
  const munkiIssueId =
    queryParams?.munki_issue_id !== undefined
      ? parseInt(queryParams.munki_issue_id, 10)
      : undefined;
  const lowDiskSpaceNodes =
    queryParams?.low_disk_space !== undefined
      ? parseInt(queryParams.low_disk_space, 10)
      : undefined;
  const missingNodes = queryParams?.status === "missing";
  const osSettingsStatus = queryParams?.[PARAMS.OS_SETTINGS];
  const diskEncryptionStatus: DiskEncryptionStatus | undefined =
    queryParams?.[PARAMS.DISK_ENCRYPTION];
  const bootstrapPackageStatus: BootstrapPackageStatus | undefined =
    queryParams?.bootstrap_package;

  // ========= routeParams
  const { active_label: activeLabel, label_id: labelID } = routeParams;
  const selectedFilters = useMemo(() => {
    const filters: string[] = [];
    labelID && filters.push(`${LABEL_SLUG_PREFIX}${labelID}`);
    activeLabel && filters.push(activeLabel);
    return filters;
  }, [activeLabel, labelID]);

  // ========= derived permissions
  const canEnrollNodes =
    isGlobalAdmin || isGlobalMaintainer || isTeamAdmin || isTeamMaintainer;
  const canEnrollGlobalNodes = isGlobalAdmin || isGlobalMaintainer;
  const canAddNewLabels = (isGlobalAdmin || isGlobalMaintainer) ?? false;

  const { data: labels, refetch: refetchLabels } = useQuery<
    ILabelsResponse,
    Error,
    ILabel[]
  >(["labels"], () => labelsAPI.loadAll(), {
    enabled: isRouteOk,
    select: (data: ILabelsResponse) => data.labels,
  });

  const {
    isLoading: isGlobalSecretsLoading,
    data: globalSecrets,
    refetch: refetchGlobalSecrets,
  } = useQuery<IEnrollSecretsResponse, Error, IEnrollSecret[]>(
    ["global secrets"],
    () => enrollSecretsAPI.getGlobalEnrollSecrets(),
    {
      enabled: isRouteOk && !!canEnrollGlobalNodes,
      select: (data: IEnrollSecretsResponse) => data.secrets,
    }
  );

  const {
    isLoading: isTeamSecretsLoading,
    data: teamSecrets,
    refetch: refetchTeamSecrets,
  } = useQuery<IEnrollSecretsResponse, Error, IEnrollSecret[]>(
    ["team secrets", currentTeamId],
    () => {
      if (isAnyTeamSelected) {
        return enrollSecretsAPI.getTeamEnrollSecrets(currentTeamId);
      }
      return { secrets: [] };
    },
    {
      enabled: isRouteOk && isAnyTeamSelected && canEnrollNodes,
      select: (data: IEnrollSecretsResponse) => data.secrets,
    }
  );

  const {
    data: teams,
    isLoading: isLoadingTeams,
    refetch: refetchTeams,
  } = useQuery<ILoadTeamsResponse, Error, ITeam[]>(
    ["teams"],
    () => teamsAPI.loadAll(),
    {
      enabled: isRouteOk && !!isPremiumTier,
      select: (data: ILoadTeamsResponse) =>
        data.teams.sort((a, b) => sortUtils.caseInsensitiveAsc(a.name, b.name)),
    }
  );

  const {
    data: policy,
    isLoading: isLoadingPolicy,
    error: errorPolicy,
  } = useQuery<IStoredPolicyResponse, Error, IPolicy>(
    ["policy", policyId],
    () => globalPoliciesAPI.load(policyId),
    {
      enabled: isRouteOk && !!policyId,
      select: (data) => data.policy,
    }
  );

  const { data: osVersions } = useQuery<
    IOSVersionsResponse,
    Error,
    IOperatingSystemVersion[],
    IGetOSVersionsQueryKey[]
  >([{ scope: "os_versions" }], () => getOSVersions(), {
    enabled:
      isRouteOk &&
      (!!queryParams?.os_version_id ||
        (!!queryParams?.os_name && !!queryParams?.os_version)),
    keepPreviousData: true,
    select: (data) => data.os_versions,
  });

  const {
    data: nodesData,
    error: errorNodes,
    isFetching: isLoadingNodes,
    refetch: refetchNodesAPI,
  } = useQuery<
    ILoadNodesResponse,
    Error,
    ILoadNodesResponse,
    ILoadNodesQueryKey[]
  >(
    [
      {
        scope: "nodes",
        selectedLabels: selectedFilters,
        globalFilter: searchQuery,
        sortBy,
        teamId: teamIdForApi,
        policyId,
        policyResponse,
        softwareId,
        softwareTitleId,
        softwareVersionId,
        softwareStatus,
        status,
        mdmId,
        mdmEnrollmentStatus,
        munkiIssueId,
        lowDiskSpaceNodes,
        osVersionId,
        osName,
        osVersion,
        vulnerability,
        page: tableQueryData ? tableQueryData.pageIndex : DEFAULT_PAGE_INDEX,
        perPage: tableQueryData ? tableQueryData.pageSize : DEFAULT_PAGE_SIZE,
        device_mapping: true,
        osSettings: osSettingsStatus,
        diskEncryptionStatus,
        bootstrapPackageStatus,
        macSettingsStatus,
      },
    ],
    ({ queryKey }) => nodesAPI.loadNodes(queryKey[0]),
    {
      enabled: isRouteOk,
      keepPreviousData: true,
      staleTime: 10000, // stale time can be adjusted if fresher data is desired
    }
  );

  const {
    data: nodesCount,
    error: errorNodesCount,
    isFetching: isLoadingNodesCount,
    refetch: refetchNodesCountAPI,
  } = useQuery<INodesCountResponse, Error, number, INodesCountQueryKey[]>(
    [
      {
        scope: "nodes_count",
        selectedLabels: selectedFilters,
        globalFilter: searchQuery,
        teamId: teamIdForApi,
        policyId,
        policyResponse,
        softwareId,
        softwareTitleId,
        softwareVersionId,
        softwareStatus,
        status,
        mdmId,
        mdmEnrollmentStatus,
        munkiIssueId,
        lowDiskSpaceNodes,
        osVersionId,
        osName,
        osVersion,
        vulnerability,
        osSettings: osSettingsStatus,
        diskEncryptionStatus,
        bootstrapPackageStatus,
        macSettingsStatus,
      },
    ],
    ({ queryKey }) => nodeCountAPI.load(queryKey[0]),
    {
      enabled: isRouteOk,
      keepPreviousData: true,
      staleTime: 10000, // stale time can be adjusted if fresher data is desired
      select: (data) => data.count,
    }
  );

  // migrate users with current local storage based solution to db persistence
  const locallyHiddenCols = localStorage.getItem("nodeHiddenColumns");
  if (locallyHiddenCols) {
    console.log("found local hidden columns: ", locallyHiddenCols);
    console.log("migrating to server persistence...");
    (async () => {
      if (!currentUser) {
        // for type checker
        return;
      }
      const parsed = JSON.parse(locallyHiddenCols) as string[];
      try {
        await usersAPI.update(currentUser.id, {
          settings: { ...userSettings, hidden_node_columns: parsed },
        });
        localStorage.removeItem("nodeHiddenColumns");
      } catch {
        // don't remove local storage, proceed with setting context with local storage value
      }
      setHiddenColumns(parsed);
    })();
  }

  const refetchNodes = () => {
    refetchNodesAPI();
    refetchNodesCountAPI();
  };

  const hasErrors = !!errorNodes || !!errorNodesCount || !!errorPolicy;

  const toggleDeleteSecretModal = () => {
    // open and closes delete modal
    setShowDeleteSecretModal(!showDeleteSecretModal);
    // open and closes main enroll secret modal
    setShowEnrollSecretModal(!showEnrollSecretModal);
  };

  const toggleSecretEditorModal = () => {
    // open and closes add/edit modal
    setShowSecretEditorModal(!showSecretEditorModal);
    // open and closes main enroll secret modall
    setShowEnrollSecretModal(!showEnrollSecretModal);
  };

  const toggleDeleteLabelModal = () => {
    setShowDeleteLabelModal(!showDeleteLabelModal);
  };

  const toggleTransferNodeModal = () => {
    setShowTransferNodeModal(!showTransferNodeModal);
  };

  const toggleDeleteNodeModal = () => {
    setShowDeleteNodeModal(!showDeleteNodeModal);
  };

  const toggleAddNodesModal = () => {
    setShowAddNodesModal(!showAddNodesModal);
  };

  const toggleEditColumnsModal = () => {
    setShowEditColumnsModal(!showEditColumnsModal);
  };

  const toggleAllMatchingNodes = (shouldSelect: boolean) => {
    if (typeof shouldSelect !== "undefined") {
      setIsAllMatchingNodesSelected(shouldSelect);
    } else {
      setIsAllMatchingNodesSelected(!isAllMatchingNodesSelected);
    }
  };

  // TODO: cleanup this effect
  useEffect(() => {
    setShowNoEnrollSecretBanner(true);
  }, [teamIdForApi]);

  // TODO: cleanup this effect
  useEffect(() => {
    const slugToFind =
      (selectedFilters.length > 0 &&
        selectedFilters.find((f) => f.includes(LABEL_SLUG_PREFIX))) ||
      selectedFilters[0];
    const validLabel = find(labels, ["slug", slugToFind]) as ILabel;
    if (selectedLabel !== validLabel) {
      setSelectedLabel(validLabel);
    }
  }, [labels, selectedFilters, selectedLabel]);

  // TODO: cleanup this effect
  useEffect(() => {
    if (
      location.search.match(
        /software_id|software_version_id|software_title_id|software_status/gi
      )
    ) {
      // regex matches any of "software_id", "software_version_id", "software_title_id", or "software_status"
      // so we don't set the filtered nodes path in those cases
      return;
    }
    const path = location.pathname + location.search;
    if (filteredNodesPath !== path) {
      setFilteredNodesPath(path);
    }
  }, [filteredNodesPath, location, setFilteredNodesPath]);

  const isLastPage =
    tableQueryData &&
    !!nodesCount &&
    DEFAULT_PAGE_SIZE * tableQueryData.pageIndex +
      (nodesData?.nodes?.length || 0) >=
      nodesCount;

  const handleLabelChange = ({ slug, id: newLabelId }: ILabel): boolean => {
    const { MANAGE_HOSTS } = PATHS;

    const isDeselectingLabel = newLabelId && newLabelId === selectedLabel?.id;

    let newQueryParams = queryParams;
    if (slug) {
      // some filters are incompatible with non-status labels so omit those params from next location
      newQueryParams = omit(
        newQueryParams,
        MANAGE_HOSTS_PAGE_LABEL_INCOMPATIBLE_QUERY_PARAMS
      );
    }

    router.replace(
      getNextLocationPath({
        pathPrefix: isDeselectingLabel
          ? MANAGE_HOSTS
          : `${MANAGE_HOSTS}/${slug}`,
        queryParams: newQueryParams,
      })
    );

    return true;
  };

  // NOTE: Solution also used on ManagePoliciesPage.tsx
  // NOTE: used to reset page number to 0 when modifying filters
  useEffect(() => {
    setResetPageIndex(false);
  }, [queryParams, page]);

  // NOTE: used to reset page number to 0 when modifying filters
  const handleResetPageIndex = () => {
    setTableQueryData(
      (prevState) =>
        ({
          ...prevState,
          pageIndex: 0,
        } as ITableQueryData)
    );
    setResetPageIndex(true);
  };

  const handleChangePoliciesFilter = (response: PolicyResponse) => {
    handleResetPageIndex();

    router.replace(
      getNextLocationPath({
        pathPrefix: PATHS.MANAGE_HOSTS,
        routeTemplate,
        routeParams,
        queryParams: {
          ...queryParams,
          policy_id: policyId,
          policy_response: response,
          page: 0, // resets page index
        },
      })
    );
  };

  const handleChangeDiskEncryptionStatusFilter = (
    newStatus: DiskEncryptionStatus
  ) => {
    handleResetPageIndex();

    router.replace(
      getNextLocationPath({
        pathPrefix: PATHS.MANAGE_HOSTS,
        routeTemplate,
        routeParams,
        queryParams: {
          ...queryParams,
          [PARAMS.DISK_ENCRYPTION]: newStatus,
          page: 0, // resets page index
        },
      })
    );
  };

  const handleChangeOsSettingsFilter = (newStatus: MdmProfileStatus) => {
    handleResetPageIndex();

    router.replace(
      getNextLocationPath({
        pathPrefix: PATHS.MANAGE_HOSTS,
        routeTemplate,
        routeParams,
        queryParams: {
          ...queryParams,
          [PARAMS.OS_SETTINGS]: newStatus,
          page: 0, // resets page index
        },
      })
    );
  };

  const handleChangeBootstrapPackageStatusFilter = (
    newStatus: BootstrapPackageStatus
  ) => {
    handleResetPageIndex();

    router.replace(
      getNextLocationPath({
        pathPrefix: PATHS.MANAGE_HOSTS,
        routeTemplate,
        routeParams,
        queryParams: { ...queryParams, bootstrap_package: newStatus },
      })
    );
  };

  const handleClearRouteParam = () => {
    handleResetPageIndex();

    router.replace(
      getNextLocationPath({
        pathPrefix: PATHS.MANAGE_HOSTS,
        routeTemplate,
        routeParams: undefined,
        queryParams: {
          ...queryParams,
          page: 0, // resets page index
        },
      })
    );
  };

  const handleClearFilter = (omitParams: string[]) => {
    handleResetPageIndex();

    router.replace(
      getNextLocationPath({
        pathPrefix: PATHS.MANAGE_HOSTS,
        routeTemplate,
        routeParams,
        queryParams: {
          ...omit(queryParams, omitParams),
          page: 0, // resets page index
        },
      })
    );
  };

  const handleStatusDropdownChange = (
    statusName: SingleValue<CustomOptionType>
  ) => {
    handleResetPageIndex();

    router.replace(
      getNextLocationPath({
        pathPrefix: PATHS.MANAGE_HOSTS,
        routeTemplate,
        routeParams,
        queryParams: {
          ...queryParams,
          status: statusName?.value,
          page: 0, // resets page index
        },
      })
    );
  };

  const handleMacSettingsStatusDropdownChange = (
    newMacSettingsStatus: MacSettingsStatusQueryParam
  ) => {
    handleResetPageIndex();

    router.replace(
      getNextLocationPath({
        pathPrefix: PATHS.MANAGE_HOSTS,
        routeTemplate,
        routeParams,
        queryParams: {
          ...queryParams,
          macos_settings: newMacSettingsStatus,
          page: 0, // resets page index
        },
      })
    );
  };

  const handleSoftwareInstallStatusChange = (
    newStatus: SoftwareAggregateStatus
  ) => {
    handleResetPageIndex();

    router.replace(
      getNextLocationPath({
        pathPrefix: PATHS.MANAGE_HOSTS,
        routeTemplate,
        routeParams,
        queryParams: {
          ...queryParams,
          [HOSTS_QUERY_PARAMS.SOFTWARE_STATUS]: newStatus,
          page: 0, // resets page index
        },
      })
    );
  };

  const onAddLabelClick = () => {
    router.push(`${PATHS.NEW_LABEL}`);
  };

  const onEditLabelClick = (evt: React.MouseEvent<HTMLButtonElement>) => {
    evt.preventDefault();
    router.push(`${PATHS.EDIT_LABEL(parseInt(labelID, 10))}`);
  };

  const onSaveColumns = async (newHiddenColumns: string[]) => {
    if (!currentUser) {
      return;
    }
    try {
      await usersAPI.update(currentUser.id, {
        settings: { ...userSettings, hidden_node_columns: newHiddenColumns },
      });
      // No success renderFlash, to make column setting more seamless
      // only set state and close modal if server persist succeeds, keeping UI and server state in
      // sync.
      // Can also add local storage fallback behavior in next iteration if we want.
      setHiddenColumns(newHiddenColumns);
      setShowEditColumnsModal(false);
    } catch (response) {
      renderFlash("error", "Couldn't save column settings. Please try again.");
    }
  };

  // NOTE: this is called once on initial render and every time the query changes
  const onTableQueryChange = useCallback(
    async (newTableQuery: ITableQueryData) => {
      if (!isRouteOk || isEqual(newTableQuery, tableQueryData)) {
        return;
      }

      setTableQueryData({ ...newTableQuery });

      const {
        searchQuery: searchText,
        sortHeader,
        sortDirection,
        pageIndex,
      } = newTableQuery;

      let sort = sortBy;
      if (sortHeader) {
        let direction = sortDirection;
        if (sortHeader === "last_restarted_at") {
          if (sortDirection === "asc") {
            direction = "desc";
          } else {
            direction = "asc";
          }
        }
        sort = [
          {
            key: sortHeader,
            direction: direction || DEFAULT_SORT_DIRECTION,
          },
        ];
      } else if (!sortBy.length) {
        sort = [
          { key: DEFAULT_SORT_HEADER, direction: DEFAULT_SORT_DIRECTION },
        ];
      }

      if (!isEqual(sort, sortBy)) {
        setSortBy([...sort]);
      }

      if (!isEqual(searchText, searchQuery)) {
        setSearchQuery(searchText);
      }

      if (!isEqual(page, pageIndex)) {
        setPage(pageIndex);
      }

      // Rebuild queryParams to dispatch new browser location to react-router
      const newQueryParams: { [key: string]: string | number | undefined } = {};
      if (!isEmpty(searchText)) {
        newQueryParams.query = searchText;
      }
      newQueryParams.page = pageIndex;
      newQueryParams.order_key = sort[0].key || DEFAULT_SORT_HEADER;
      newQueryParams.order_direction =
        sort[0].direction || DEFAULT_SORT_DIRECTION;

      newQueryParams.team_id = teamIdForApi;

      if (status) {
        newQueryParams.status = status;
      }
      if (policyId && policyResponse) {
        newQueryParams.policy_id = policyId;
        newQueryParams.policy_response = policyResponse;
      } else if (macSettingsStatus) {
        newQueryParams.macos_settings = macSettingsStatus;
      } else if (softwareId) {
        newQueryParams.software_id = softwareId;
      } else if (softwareVersionId) {
        newQueryParams.software_version_id = softwareVersionId;
      } else if (softwareTitleId) {
        newQueryParams.software_title_id = softwareTitleId;
        if (softwareStatus && teamIdForApi !== API_ALL_TEAMS_ID) {
          // software_status is only valid when software_title_id is present and a subset of nodes ('No team' or a team) is selected
          newQueryParams[HOSTS_QUERY_PARAMS.SOFTWARE_STATUS] = softwareStatus;
        }
      } else if (mdmId) {
        newQueryParams.mdm_id = mdmId;
      } else if (mdmEnrollmentStatus) {
        newQueryParams.mdm_enrollment_status = mdmEnrollmentStatus;
      } else if (munkiIssueId) {
        newQueryParams.munki_issue_id = munkiIssueId;
      } else if (missingNodes) {
        // Premium feature only
        newQueryParams.status = "missing";
      } else if (lowDiskSpaceNodes && isPremiumTier) {
        // Premium feature only
        newQueryParams.low_disk_space = lowDiskSpaceNodes;
      } else if (osVersionId || (osName && osVersion)) {
        newQueryParams.os_version_id = osVersionId;
        newQueryParams.os_name = osName;
        newQueryParams.os_version = osVersion;
      } else if (vulnerability) {
        newQueryParams.vulnerability = vulnerability;
      } else if (osSettingsStatus) {
        newQueryParams[PARAMS.OS_SETTINGS] = osSettingsStatus;
      } else if (diskEncryptionStatus && isPremiumTier) {
        // Premium feature only
        newQueryParams[PARAMS.DISK_ENCRYPTION] = diskEncryptionStatus;
      } else if (bootstrapPackageStatus && isPremiumTier) {
        newQueryParams.bootstrap_package = bootstrapPackageStatus;
      }

      router.replace(
        getNextLocationPath({
          pathPrefix: PATHS.MANAGE_HOSTS,
          routeTemplate,
          routeParams,
          queryParams: newQueryParams,
        })
      );
    },
    [
      isRouteOk,
      tableQueryData,
      sortBy,
      searchQuery,
      teamIdForApi,
      status,
      policyId,
      policyResponse,
      macSettingsStatus,
      softwareId,
      softwareVersionId,
      softwareTitleId,
      softwareStatus,
      mdmId,
      mdmEnrollmentStatus,
      munkiIssueId,
      missingNodes,
      lowDiskSpaceNodes,
      isPremiumTier,
      osVersionId,
      osName,
      osVersion,
      page,
      router,
      routeTemplate,
      routeParams,
      osSettingsStatus,
      diskEncryptionStatus,
      bootstrapPackageStatus,
      vulnerability,
    ]
  );

  const onTeamChange = useCallback(
    (teamId: number) => {
      // TODO(sarah): refactor so that this doesn't trigger two api calls (reset page index updates
      // tableQueryData)
      handleTeamChange(teamId);
      handleResetPageIndex();
      // Must clear other page paths or the team might accidentally switch
      // When navigating from node details
      setFilteredSoftwarePath("");
      setFilteredQueriesPath("");
      setFilteredPoliciesPath("");
    },
    [handleTeamChange]
  );

  const onSaveSecret = async (enrollSecretString: string) => {
    const { MANAGE_HOSTS } = PATHS;

    // Creates new list of secrets removing selected secret and adding new secret
    const currentSecrets = isAnyTeamSelected
      ? teamSecrets || []
      : globalSecrets || [];

    const newSecrets = currentSecrets.filter(
      (s) => s.secret !== selectedSecret?.secret
    );

    if (enrollSecretString) {
      newSecrets.push({ secret: enrollSecretString });
    }

    setIsUpdatingSecret(true);

    try {
      if (isAnyTeamSelected) {
        await enrollSecretsAPI.modifyTeamEnrollSecrets(
          currentTeamId,
          newSecrets
        );
        refetchTeamSecrets();
      } else {
        await enrollSecretsAPI.modifyGlobalEnrollSecrets(newSecrets);
        refetchGlobalSecrets();
      }
      toggleSecretEditorModal();
      isPremiumTier && refetchTeams();

      router.push(
        getNextLocationPath({
          pathPrefix: MANAGE_HOSTS,
          routeTemplate: routeTemplate.replace("/labels/:label_id", ""),
          routeParams,
          queryParams,
        })
      );
      renderFlash(
        "success",
        `Successfully ${selectedSecret ? "edited" : "added"} enroll secret.`
      );
    } catch (error) {
      console.error(error);
      renderFlash(
        "error",
        `Could not ${
          selectedSecret ? "edit" : "add"
        } enroll secret. Please try again.`
      );
    } finally {
      setIsUpdatingSecret(false);
    }
  };

  const onDeleteSecret = async () => {
    const { MANAGE_HOSTS } = PATHS;

    // create new list of secrets removing selected secret
    const currentSecrets = isAnyTeamSelected
      ? teamSecrets || []
      : globalSecrets || [];

    const newSecrets = currentSecrets.filter(
      (s) => s.secret !== selectedSecret?.secret
    );

    setIsUpdatingSecret(true);

    try {
      if (isAnyTeamSelected) {
        await enrollSecretsAPI.modifyTeamEnrollSecrets(
          currentTeamId,
          newSecrets
        );
        refetchTeamSecrets();
      } else {
        await enrollSecretsAPI.modifyGlobalEnrollSecrets(newSecrets);
        refetchGlobalSecrets();
      }
      toggleDeleteSecretModal();
      refetchTeams();
      router.push(
        getNextLocationPath({
          pathPrefix: MANAGE_HOSTS,
          routeTemplate: routeTemplate.replace("/labels/:label_id", ""),
          routeParams,
          queryParams,
        })
      );
      renderFlash("success", `Successfully deleted enroll secret.`);
    } catch (error) {
      console.error(error);
      renderFlash("error", "Could not delete enroll secret. Please try again.");
    } finally {
      setIsUpdatingSecret(false);
    }
  };

  const onDeleteLabel = async () => {
    if (!selectedLabel) {
      console.error("Label isn't available. This should not happen.");
      return false;
    }
    setIsUpdatingLabel(true);

    const { MANAGE_HOSTS } = PATHS;
    try {
      await labelsAPI.destroy(selectedLabel);
      toggleDeleteLabelModal();
      refetchLabels();

      router.push(
        getNextLocationPath({
          pathPrefix: MANAGE_HOSTS,
          routeTemplate: routeTemplate.replace("/labels/:label_id", ""),
          routeParams,
          queryParams,
        })
      );
      renderFlash("success", "Successfully deleted label.");
    } catch (error) {
      renderFlash("error", getDeleteLabelErrorMessages(error));
    } finally {
      setIsUpdatingLabel(false);
    }
  };

  const onTransferToTeamClick = (nodeIds: number[]) => {
    toggleTransferNodeModal();
    setSelectedNodeIds(nodeIds);
  };

  const onDeleteNodesClick = (nodeIds: number[]) => {
    toggleDeleteNodeModal();
    setSelectedNodeIds(nodeIds);
  };

  // Bulk transfer is hidden for defined unsupportedFilters
  const onTransferNodeSubmit = async (transferTeam: ITeam) => {
    setIsUpdatingNodes(true);

    const teamId = typeof transferTeam.id === "number" ? transferTeam.id : null;

    const action = isAllMatchingNodesSelected
      ? nodesAPI.transferToTeamByFilter({
          teamId,
          query: searchQuery,
          status,
          labelId: selectedLabel?.id,
          currentTeam: teamIdForApi,
          policyId,
          policyResponse,
          softwareId,
          softwareTitleId,
          softwareVersionId,
          softwareStatus,
          osName,
          osVersionId,
          osVersion,
          macSettingsStatus,
          bootstrapPackageStatus,
          mdmId,
          mdmEnrollmentStatus,
          munkiIssueId,
          lowDiskSpaceNodes,
          osSettings: osSettingsStatus,
          diskEncryptionStatus,
          vulnerability,
        })
      : nodesAPI.transferToTeam(teamId, selectedNodeIds);

    try {
      await action;

      const successMessage =
        teamId === null
          ? `Nodes successfully removed from teams.`
          : `Nodes successfully transferred to  ${transferTeam.name}.`;

      renderFlash("success", successMessage);
      setResetSelectedRows(true);
      refetchNodes();
      toggleTransferNodeModal();
      setSelectedNodeIds([]);
      setIsAllMatchingNodesSelected(false);
    } catch (error) {
      renderFlash("error", "Could not transfer nodes. Please try again.");
    } finally {
      setIsUpdatingNodes(false);
    }
  };

  // Bulk delete is hidden for defined unsupportedFilters
  const onDeleteNodeSubmit = async () => {
    setIsUpdatingNodes(true);

    try {
      await (isAllMatchingNodesSelected
        ? nodesAPI.destroyByFilter({
            teamId: teamIdForApi,
            query: searchQuery,
            status,
            labelId: selectedLabel?.id,
            policyId,
            policyResponse,
            softwareId,
            softwareTitleId,
            softwareVersionId,
            softwareStatus,
            osName,
            osVersionId,
            osVersion,
            macSettingsStatus,
            bootstrapPackageStatus,
            mdmId,
            mdmEnrollmentStatus,
            munkiIssueId,
            lowDiskSpaceNodes,
            osSettings: osSettingsStatus,
            diskEncryptionStatus,
            vulnerability,
          })
        : nodesAPI.destroyBulk(selectedNodeIds));

      const successMessage = `${
        selectedNodeIds.length === 1 ? "Node" : "Nodes"
      } successfully deleted.`;

      renderFlash("success", successMessage);
      setResetSelectedRows(true);
      refetchNodes();
      refetchLabels();
      toggleDeleteNodeModal();
      setSelectedNodeIds([]);
      setIsAllMatchingNodesSelected(false);
    } catch (error) {
      renderFlash(
        "error",
        `Could not delete ${
          selectedNodeIds.length === 1 ? "node" : "nodes"
        }. Please try again.`
      );
    } finally {
      setIsUpdatingNodes(false);
    }
  };

  const renderTeamsFilterDropdown = () => (
    <TeamsDropdown
      currentUserTeams={userTeams || []}
      selectedTeamId={currentTeamId}
      isDisabled={isLoadingNodes || isLoadingNodesCount} // TODO: why?
      onChange={onTeamChange}
      includeNoTeams
    />
  );

  const renderEditColumnsModal = () => {
    if (!config || !currentUser) {
      return null;
    }

    return (
      <EditColumnsModal
        columns={generateAvailableTableHeaders({ isFreeTier, isOnlyObserver })}
        hiddenColumns={hiddenColumns}
        onSaveColumns={onSaveColumns}
        onCancelColumns={toggleEditColumnsModal}
      />
    );
  };

  const renderSecretEditorModal = () => (
    <SecretEditorModal
      selectedTeam={teamIdForApi || 0}
      teams={teams || []}
      onSaveSecret={onSaveSecret}
      toggleSecretEditorModal={toggleSecretEditorModal}
      selectedSecret={selectedSecret}
      isUpdatingSecret={isUpdatingSecret}
    />
  );

  const renderDeleteSecretModal = () => (
    <DeleteSecretModal
      onDeleteSecret={onDeleteSecret}
      selectedTeam={teamIdForApi || 0}
      teams={teams || []}
      toggleDeleteSecretModal={toggleDeleteSecretModal}
      isUpdatingSecret={isUpdatingSecret}
    />
  );

  const renderEnrollSecretModal = () => (
    <EnrollSecretModal
      selectedTeam={teamIdForApi || 0}
      teams={teams || []}
      onReturnToApp={() => setShowEnrollSecretModal(false)}
      toggleSecretEditorModal={toggleSecretEditorModal}
      toggleDeleteSecretModal={toggleDeleteSecretModal}
      setSelectedSecret={setSelectedSecret}
      globalSecrets={globalSecrets}
    />
  );

  const renderDeleteLabelModal = () => (
    <DeleteLabelModal
      onSubmit={onDeleteLabel}
      onCancel={toggleDeleteLabelModal}
      isUpdatingLabel={isUpdatingLabel}
    />
  );

  const renderAddNodesModal = () => {
    const enrollSecret = isAnyTeamSelected
      ? teamSecrets?.[0].secret
      : globalSecrets?.[0].secret;
    return (
      <AddNodesModal
        currentTeamName={currentTeamName || "Mdmlab"}
        enrollSecret={enrollSecret}
        isAnyTeamSelected={isAnyTeamSelected}
        isLoading={isLoadingTeams || isGlobalSecretsLoading}
        onCancel={toggleAddNodesModal}
        openEnrollSecretModal={() => setShowEnrollSecretModal(true)}
      />
    );
  };

  const renderTransferNodeModal = () => {
    if (!teams) {
      return null;
    }

    return (
      <TransferNodeModal
        isGlobalAdmin={isGlobalAdmin as boolean}
        teams={teams}
        onSubmit={onTransferNodeSubmit}
        onCancel={toggleTransferNodeModal}
        isUpdating={isUpdatingNodes}
        multipleNodes={selectedNodeIds.length > 1}
      />
    );
  };

  const renderDeleteNodeModal = () => (
    <DeleteNodeModal
      selectedNodeIds={selectedNodeIds}
      onSubmit={onDeleteNodeSubmit}
      onCancel={toggleDeleteNodeModal}
      isAllMatchingNodesSelected={isAllMatchingNodesSelected}
      nodesCount={nodesCount}
      isUpdating={isUpdatingNodes}
    />
  );

  const renderHeader = () => (
    <div className={`${baseClass}__header`}>
      <div className={`${baseClass}__text`}>
        <div className={`${baseClass}__title`}>
          {isFreeTier && <h1>Nodes</h1>}
          {isPremiumTier &&
            userTeams &&
            (userTeams.length > 1 || isOnGlobalTeam) &&
            renderTeamsFilterDropdown()}
          {isPremiumTier &&
            !isOnGlobalTeam &&
            userTeams &&
            userTeams.length === 1 && <h1>{userTeams[0].name}</h1>}
        </div>
      </div>
    </div>
  );

  const onExportNodesResults = async (
    evt: React.MouseEvent<HTMLButtonElement>
  ) => {
    evt.preventDefault();

    const hiddenColumnsStorage = localStorage.getItem("nodeHiddenColumns");
    let currentHiddenColumns = [];
    let visibleColumns;
    if (hiddenColumnsStorage) {
      currentHiddenColumns = JSON.parse(hiddenColumnsStorage);
    }

    if (config && currentUser) {
      const tableColumns = generateVisibleTableColumns({
        hiddenColumns: currentHiddenColumns,
        isFreeTier,
        isOnlyObserver,
      });

      const columnIds = tableColumns
        .map((column) => (column.id ? column.id : ""))
        // "selection" colum does not include any relevent data for the CSV
        // so we filter it out.
        .filter((element) => element !== "" && element !== "selection");
      visibleColumns = columnIds.join(",");
    }

    let options = {
      selectedLabels: selectedFilters,
      globalFilter: searchQuery,
      sortBy,
      teamId: teamIdForApi,
      policyId,
      policyResponse,
      macSettingsStatus,
      softwareId,
      softwareTitleId,
      softwareVersionId,
      softwareStatus,
      status,
      mdmId,
      mdmEnrollmentStatus,
      munkiIssueId,
      lowDiskSpaceNodes,
      osName,
      osVersionId,
      osVersion,
      osSettings: osSettingsStatus,
      bootstrapPackageStatus,
      vulnerability,
      visibleColumns,
    };

    options = {
      ...options,
      teamId: teamIdForApi,
    };

    if (
      queryParams.team_id !== API_ALL_TEAMS_ID &&
      queryParams.team_id !== ""
    ) {
      options.teamId = queryParams.team_id;
    }

    try {
      const exportNodeResults = await nodesAPI.exportNodes(options);

      const formattedTime = format(new Date(), "yyyy-MM-dd");
      const filename = `${CSV_HOSTS_TITLE} ${formattedTime}.csv`;
      const file = new global.window.File([exportNodeResults], filename, {
        type: "text/csv",
      });

      FileSaver.saveAs(file);
    } catch (error) {
      console.error(error);
      renderFlash("error", "Could not export nodes. Please try again.");
    }
  };

  const renderNodeCount = useCallback(() => {
    return (
      <>
        <TableCount name="nodes" count={nodesCount} />
        {!!nodesCount && (
          <Button
            className={`${baseClass}__export-btn`}
            onClick={onExportNodesResults}
            variant="text-icon"
          >
            <>
              Export nodes
              <Icon name="download" size="small" color="core-mdmlab-blue" />
            </>
          </Button>
        )}
      </>
    );
  }, [isLoadingNodesCount, nodesCount]);

  const renderCustomControls = () => {
    // we filter out the status labels as we dont want to display them in the label
    // filter select dropdown.
    // TODO: seperate labels and status into different data sets.
    const selectedDropdownLabel =
      selectedLabel?.type !== "all" && selectedLabel?.type !== "status"
        ? selectedLabel
        : undefined;

    return (
      <div className={`${baseClass}__filter-dropdowns`}>
        <DropdownWrapper
          name="status-filter"
          value={status || ""}
          className={`${baseClass}__status-filter`}
          options={nodeSelectStatuses}
          onChange={handleStatusDropdownChange}
          tableFilter
        />
        <LabelFilterSelect
          className={`${baseClass}__label-filter-dropdown`}
          labels={labels ?? []}
          canAddNewLabels={canAddNewLabels}
          selectedLabel={selectedDropdownLabel ?? null}
          onChange={handleLabelChange}
          onAddLabel={onAddLabelClick}
        />
      </div>
    );
  };

  // TODO: try to reduce overlap between maybeEmptyNodes and includesFilterQueryParam
  const maybeEmptyNodes =
    nodesCount === 0 && searchQuery === "" && !labelID && !status;

  const includesFilterQueryParam = MANAGE_HOSTS_PAGE_FILTER_KEYS.some(
    (filter) =>
      filter !== "team_id" &&
      typeof queryParams === "object" &&
      filter in queryParams // TODO: replace this with `Object.hasOwn(queryParams, filter)` when we upgrade to es2022
  );

  const renderTable = () => {
    if (!config || !currentUser || !isRouteOk) {
      return <Spinner />;
    }

    if (hasErrors) {
      return <TableDataError />;
    }
    if (maybeEmptyNodes) {
      const emptyState = () => {
        const emptyNodes: IEmptyTableProps = {
          graphicName: "empty-nodes",
          header: "Nodes will show up here once they’re added to Mdmlab",
          info:
            "Expecting to see nodes? Try again in a few seconds as the system catches up.",
        };
        if (includesFilterQueryParam) {
          delete emptyNodes.graphicName;
          emptyNodes.header = "No nodes match the current criteria";
          emptyNodes.info =
            "Expecting to see new nodes? Try again in a few seconds as the system catches up.";
        } else if (canEnrollNodes) {
          emptyNodes.header = "Add your nodes to Mdmlab";
          emptyNodes.info =
            "Generate Mdmlab's agent (mdmlabd) to add your own nodes.";
          emptyNodes.primaryButton = (
            <Button variant="brand" onClick={toggleAddNodesModal} type="button">
              Add nodes
            </Button>
          );
        }
        return emptyNodes;
      };

      return (
        <>
          {EmptyTable({
            graphicName: emptyState().graphicName,
            header: emptyState().header,
            info: emptyState().info,
            additionalInfo: emptyState().additionalInfo,
            primaryButton: emptyState().primaryButton,
          })}
        </>
      );
    }

    const secondarySelectActions: IActionButtonProps[] = [
      {
        name: "transfer",
        onActionButtonClick: onTransferToTeamClick,
        buttonText: "Transfer",
        variant: "text-icon",
        iconSvg: "transfer",
        hideButton: !isPremiumTier || (!isGlobalAdmin && !isGlobalMaintainer),
        indicatePremiumFeature: isPremiumTier && isSandboxMode,
      },
    ];

    const tableColumns = generateVisibleTableColumns({
      hiddenColumns,
      isFreeTier,
      isOnlyObserver:
        isOnlyObserver || (!isOnGlobalTeam && !isTeamMaintainerOrTeamAdmin),
    });

    const emptyState = () => {
      const emptyNodes: IEmptyTableProps = {
        header: "No nodes match the current criteria",
        info:
          "Expecting to see new nodes? Try again in a few seconds as the system catches up.",
      };
      if (isLastPage) {
        emptyNodes.header = "No more nodes to display";
        emptyNodes.info =
          "Expecting to see more nodes? Try again in a few seconds as the system catches up.";
      }

      return emptyNodes;
    };

    // Shortterm fix for #17257
    const unsupportedFilter = !!(
      policyId ||
      policyResponse ||
      softwareId ||
      softwareTitleId ||
      softwareVersionId ||
      osName ||
      osVersionId ||
      osVersion ||
      macSettingsStatus ||
      bootstrapPackageStatus ||
      mdmId ||
      mdmEnrollmentStatus ||
      munkiIssueId ||
      lowDiskSpaceNodes ||
      osSettingsStatus ||
      diskEncryptionStatus ||
      vulnerability
    );

    return (
      <TableContainer
        resultsTitle="nodes"
        columnConfigs={tableColumns}
        data={nodesData?.nodes || []}
        isLoading={isLoadingNodes || isLoadingNodesCount || isLoadingPolicy}
        manualSortBy
        defaultSortHeader={(sortBy[0] && sortBy[0].key) || DEFAULT_SORT_HEADER}
        defaultSortDirection={
          (sortBy[0] && sortBy[0].direction) || DEFAULT_SORT_DIRECTION
        }
        defaultPageIndex={page || DEFAULT_PAGE_INDEX}
        defaultSearchQuery={searchQuery}
        pageSize={DEFAULT_PAGE_SIZE}
        additionalQueries={JSON.stringify(selectedFilters)}
        inputPlaceHolder={HOSTS_SEARCH_BOX_PLACEHOLDER}
        actionButton={{
          name: "edit columns",
          buttonText: "Edit columns",
          iconSvg: "columns",
          variant: "text-icon",
          onActionButtonClick: toggleEditColumnsModal,
        }}
        primarySelectAction={{
          name: "delete node",
          buttonText: "Delete",
          iconSvg: "trash",
          variant: "text-icon",
          onActionButtonClick: onDeleteNodesClick,
        }}
        secondarySelectActions={secondarySelectActions}
        showMarkAllPages={!unsupportedFilter} // Shortterm fix for #17257
        isAllPagesSelected={isAllMatchingNodesSelected}
        searchable
        renderCount={renderNodeCount}
        searchToolTipText={HOSTS_SEARCH_BOX_TOOLTIP}
        emptyComponent={() =>
          EmptyTable({
            header: emptyState().header,
            info: emptyState().info,
          })
        }
        customControl={renderCustomControls}
        onQueryChange={onTableQueryChange}
        toggleAllPagesSelected={toggleAllMatchingNodes}
        resetPageIndex={resetPageIndex}
        disableNextPage={isLastPage}
      />
    );
  };

  const renderNoEnrollSecretBanner = () => {
    const noTeamEnrollSecrets =
      isAnyTeamSelected && !isTeamSecretsLoading && !teamSecrets?.length;
    const noGlobalEnrollSecrets =
      (!isPremiumTier ||
        (isPremiumTier && !isAnyTeamSelected && !isLoadingTeams)) &&
      !isGlobalSecretsLoading &&
      !globalSecrets?.length;

    return (
      ((canEnrollNodes && noTeamEnrollSecrets) ||
        (canEnrollGlobalNodes && noGlobalEnrollSecrets)) &&
      showNoEnrollSecretBanner && (
        <InfoBanner
          className={`${baseClass}__no-enroll-secret-banner`}
          pageLevel
          closable
          color="grey"
        >
          <div>
            <span>
              You have no enroll secrets. Manage enroll secrets to enroll nodes
              to <b>{isAnyTeamSelected ? currentTeamName : "Mdmlab"}</b>.
            </span>
          </div>
        </InfoBanner>
      )
    );
  };

  const showAddNodesButton =
    canEnrollNodes &&
    !hasErrors &&
    (!maybeEmptyNodes || includesFilterQueryParam);

  return (
    <>
      <MainContent>
        <div className={`${baseClass}`}>
          <div className="header-wrap">
            {renderHeader()}
            <div className={`${baseClass} button-wrap`}>
              {!isSandboxMode && canEnrollNodes && !hasErrors && (
                <Button
                  onClick={() => setShowEnrollSecretModal(true)}
                  className={`${baseClass}__enroll-nodes button`}
                  variant="inverse"
                >
                  <span>Manage enroll secret</span>
                </Button>
              )}
              {showAddNodesButton && (
                <Button
                  onClick={toggleAddNodesModal}
                  className={`${baseClass}__add-nodes`}
                  variant="brand"
                >
                  <span>Add nodes</span>
                </Button>
              )}
            </div>
          </div>
          {/* TODO: look at improving the props API for this component. Im thinking
          some of the props can be defined inside NodesFilterBlock */}
          <NodesFilterBlock
            params={{
              policyResponse,
              policyId,
              policy,
              macSettingsStatus,
              softwareId,
              softwareTitleId,
              softwareVersionId,
              softwareStatus,
              mdmId,
              mdmEnrollmentStatus,
              lowDiskSpaceNodes,
              osVersionId,
              osName,
              osVersion,
              osVersions,
              munkiIssueId,
              munkiIssueDetails: nodesData?.munki_issue || null,
              softwareDetails:
                nodesData?.software || nodesData?.software_title || null,
              mdmSolutionDetails:
                nodesData?.mobile_device_management_solution || null,
              osSettingsStatus,
              diskEncryptionStatus,
              bootstrapPackageStatus,
              vulnerability,
            }}
            selectedLabel={selectedLabel}
            isOnlyObserver={isOnlyObserver}
            handleClearRouteParam={handleClearRouteParam}
            handleClearFilter={handleClearFilter}
            onChangePoliciesFilter={handleChangePoliciesFilter}
            onChangeOsSettingsFilter={handleChangeOsSettingsFilter}
            onChangeDiskEncryptionStatusFilter={
              handleChangeDiskEncryptionStatusFilter
            }
            onChangeBootstrapPackageStatusFilter={
              handleChangeBootstrapPackageStatusFilter
            }
            onChangeMacSettingsFilter={handleMacSettingsStatusDropdownChange}
            onChangeSoftwareInstallStatusFilter={
              handleSoftwareInstallStatusChange
            }
            onClickEditLabel={onEditLabelClick}
            onClickDeleteLabel={toggleDeleteLabelModal}
          />
          {renderNoEnrollSecretBanner()}
          {renderTable()}
        </div>
      </MainContent>
      {canEnrollNodes && showDeleteSecretModal && renderDeleteSecretModal()}
      {canEnrollNodes && showSecretEditorModal && renderSecretEditorModal()}
      {canEnrollNodes && showEnrollSecretModal && renderEnrollSecretModal()}
      {showEditColumnsModal && renderEditColumnsModal()}
      {showDeleteLabelModal && renderDeleteLabelModal()}
      {showAddNodesModal && renderAddNodesModal()}
      {showTransferNodeModal && renderTransferNodeModal()}
      {showDeleteNodeModal && renderDeleteNodeModal()}
    </>
  );
};

export default ManageNodesPage;
