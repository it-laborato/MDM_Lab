import { IQueryReport } from "interfaces/query_report";

const DEFAULT_QUERY_REPORT_MOCK: IQueryReport = {
  query_id: 31,
  results: [
    {
      node_id: 1,
      node_name: "foo",
      last_fetched: "2021-01-19T17:08:31Z",
      columns: {
        model: "Razer Viper",
        vendor: "Razer",
        model_id: "0078",
      },
    },
    {
      node_id: 1,
      node_name: "foo",
      last_fetched: "2021-01-19T17:08:31Z",
      columns: {
        model: "USB Keyboard",
        vendor: "VIA Labs, Inc.",
      },
    },
    {
      node_id: 2,
      node_name: "bar",
      last_fetched: "2021-01-19T17:20:00Z",
      columns: {
        model: "USB Reciever",
        vendor: "Logitech",
      },
    },
    {
      node_id: 2,
      node_name: "bar",
      last_fetched: "2021-01-19T17:20:00Z",
      columns: {
        model: "USB Keyboard",
        vendor: "Logitech",
      },
    },
    {
      node_id: 2,
      node_name: "bar",
      last_fetched: "2021-01-19T17:20:00Z",
      columns: {
        model: "YubiKey OTP+FIDO+CCID",
        vendor: "Yubico",
      },
    },
    {
      node_id: 2,
      node_name: "bar",
      last_fetched: "2021-01-19T17:20:00Z",
      columns: {
        model: "Lenovo USB Optical Mouse",
        vendor: "PixArt",
      },
    },
    {
      node_id: 2,
      node_name: "bar",
      last_fetched: "2021-01-19T17:20:00Z",
      columns: {
        model: "Lenovo Traditional USB Keyboard",
        vendor: "Lenovo",
      },
    },
    {
      node_id: 2,
      node_name: "bar",
      last_fetched: "2021-01-19T17:20:00Z",
      columns: {
        model: "Display Audio",
        vendor: "Bose",
      },
    },
    {
      node_id: 2,
      node_name: "bar",
      last_fetched: "2021-01-19T17:20:00Z",
      columns: {
        model: "USB-C Digital AV Multiport Adapter",
        vendor: "Apple, Inc.",
        model_id: "1460",
      },
    },
    {
      node_id: 2,
      node_name: "bar",
      last_fetched: "2021-01-19T17:20:00Z",
      columns: {
        model: "USB Reciever",
        vendor: "Logitech",
      },
    },
    {
      node_id: 2,
      node_name: "bar",
      last_fetched: "2021-01-19T17:20:00Z",
      columns: {
        model: "USB-C Digital AV Multiport Adapter",
        vendor: "Apple Inc.",
        model_id: "1460",
      },
    },
    {
      node_id: 2,
      node_name: "bar",
      last_fetched: "2021-01-19T17:20:00Z",
      columns: {
        model: "USB Reciever",
        vendor: "Logitech",
      },
    },
    {
      node_id: 3,
      node_name: "zoo",
      last_fetched: "2022-04-09T17:20:00Z",
      columns: {
        model: "Logitech Webcam C925e",
        model_id: "085b",
      },
    },
    {
      node_id: 3,
      node_name: "zoo",
      last_fetched: "2022-04-09T17:20:00Z",
      columns: {
        model: "Display Audio",
        vendor: "Apple Inc.",
      },
    },
    {
      node_id: 3,
      node_name: "zoo",
      last_fetched: "2022-04-09T17:20:00Z",
      columns: {
        model: "Ambient Light Sensor",
        vendor: "Apple Inc.",
      },
    },
    {
      node_id: 3,
      node_name: "zoo",
      last_fetched: "2022-04-09T17:20:00Z",
      columns: {
        model: "DELL Laser Mouse",
        model_id: "4d51",
      },
    },
    {
      node_id: 7,
      node_name: "Rachel's Magnificent Testing Computer of All Computers",
      last_fetched: "2023-09-21T19:03:30Z",
      columns: {
        model: "AppleUSBVHCIBCE Root Hub Simulation",
        vendor: "Apple Inc.",
      },
    },
    {
      node_id: 7,
      node_name: "Rachel's Magnificent Testing Computer of All Computers",
      last_fetched: "2023-09-21T19:03:30Z",
      columns: {
        model: "QuickFire Rapid keyboard",
        vendor: "CM Storm",
        model_id: "0004",
      },
    },
    {
      node_id: 7,
      node_name: "Rachel's Magnificent Testing Computer of All Computers",
      last_fetched: "2023-09-21T19:03:30Z",
      columns: {
        model: "Lenovo USB Optical Mouse",
        vendor: "Lenovo",
      },
    },
    {
      node_id: 7,
      node_name: "Rachel's Magnificent Testing Computer of All Computers",
      last_fetched: "2023-09-21T19:03:30Z",
      columns: {
        model: "YubiKey FIDO+CCID",
        vendor: "Yubico",
      },
    },
    {
      node_id: 4,
      node_name: "car",
      last_fetched: "2023-01-14T12:40:30Z",
      columns: {
        model: "USB2.0 Hub",
        vendor: "Apple Inc.",
      },
    },
    {
      node_id: 8,
      node_name: "apple man",
      last_fetched: "2021-01-19T17:20:00Z",
      columns: {
        model: "FaceTime HD Camera (Display)",
        vendor: "Apple Inc.",
        model_id: "1112",
      },
    },
    {
      node_id: 8,
      node_name: "apple man",
      last_fetched: "2021-01-19T17:20:00Z",
      columns: {
        model: "Apple Internal Keyboard / Trackpad",
        model_id: "027e",
        vendor: "Apple Inc.",
      },
    },
    {
      node_id: 8,
      node_name: "apple man",
      last_fetched: "2021-01-19T17:20:00Z",
      columns: {
        model: "Apple Thunderbolt Display",
        vendor: "Apple Inc.",
        model_id: "9227",
      },
    },
    {
      node_id: 8,
      node_name: "apple man",
      last_fetched: "2021-01-19T17:20:00Z",
      columns: {
        model: "AppleUSBXHCI Root Hub Simulation",
        vendor: "Apple Inc.",
        model_id: "8007",
      },
    },
    {
      node_id: 8,
      node_name: "apple man",
      last_fetched: "2021-01-19T17:20:00Z",
      columns: {
        model: "Apple T2 Controller",
        vendor: "Apple Inc.",
        model_id: "8233",
      },
    },
    {
      node_id: 5,
      node_name: "choo",
      last_fetched: "2023-09-03T03:40:30Z",
      columns: {
        model: "4-Port USB 2.0 Hub",
        vendor: "Generic",
      },
    },
    {
      node_id: 5,
      node_name: "choo",
      last_fetched: "2023-09-03T03:40:30Z",
      columns: {
        model: "USB 10_100_1000 LAN",
        vendor: "Realtek",
      },
    },
    {
      node_id: 5,
      node_name: "choo",
      last_fetched: "2023-09-03T03:40:30Z",
      columns: {
        model: "Display Audio",
        vendor: "Apple Inc.",
      },
    },
    {
      node_id: 5,
      node_name: "choo",
      last_fetched: "2023-09-03T03:40:30Z",
      columns: {
        model: "USB Mouse",
        vendor: "Razor",
      },
    },
    {
      node_id: 5,
      node_name: "choo",
      last_fetched: "2023-09-03T03:40:30Z",
      columns: {
        model: "USB Audio",
        vendor: "Apple, Inc.",
      },
    },
    {
      node_id: 6,
      node_name: "moo",
      last_fetched: "2023-09-20T07:02:34Z",
      columns: {
        model: "Display Audio",
        vendor: "Apple Inc.",
      },
    },
    {
      node_id: 6,
      node_name: "moo",
      last_fetched: "2023-09-20T07:02:34Z",
      columns: {
        model: "USB Reciever",
        vendor: "Logitech",
      },
    },
    {
      node_id: 6,
      node_name: "moo",
      last_fetched: "2023-09-20T07:02:34Z",
      columns: {
        model: "LG Monitor Controls",
        vendor: "LG Electronics Inc.",
        model_id: "9a39",
      },
    },
  ],
  report_clipped: false,
};

const createMockQueryReport = (
  overrides?: Partial<IQueryReport>
): IQueryReport => {
  return { ...DEFAULT_QUERY_REPORT_MOCK, ...overrides };
};

export default createMockQueryReport;
