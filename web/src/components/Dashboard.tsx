"use client";

import { useState, useEffect } from "react";
import { api } from "@/lib/api";

interface DashboardProps {
  onLogout: () => void;
}

interface Wallet {
  id: string;
  name: string;
  type: "hot" | "warm" | "cold";
  coin: string;
  balance: string;
  status: "active" | "pending" | "disabled";
  bitgoWalletId?: string;
}

interface Transfer {
  id: string;
  walletName: string;
  type: "warm" | "cold";
  amount: string;
  coin: string;
  destination: string;
  status:
    | "draft"
    | "submitted"
    | "pending_approval"
    | "signed"
    | "broadcast"
    | "confirmed"
    | "failed";
  createdAt: string;
  completedAt?: string;
}

interface CreateWalletForm {
  name: string;
  type: "warm" | "cold";
  coin: string;
  passphrase: string;
}

export const Dashboard: React.FC<DashboardProps> = ({ onLogout }) => {
  const [activeTab, setActiveTab] = useState<
    "overview" | "wallets" | "transfers" | "new-transfer" | "create-wallet"
  >("overview");
  const [selectedWalletType, setSelectedWalletType] = useState<"warm" | "cold">(
    "warm"
  );
  const [wallets, setWallets] = useState<Wallet[]>([]);
  const [transfers, setTransfers] = useState<Transfer[]>([]);
  const [loading, setLoading] = useState(false);
  const [createWalletForm, setCreateWalletForm] = useState<CreateWalletForm>({
    name: "",
    type: "warm",
    coin: "BTC",
    passphrase: "",
  });
  const [error, setError] = useState<string | null>(null);

  // Load data from API
  useEffect(() => {
    loadWallets();
    loadTransfers();
  }, []);

  const loadWallets = async () => {
    try {
      setLoading(true);
      const response = await api.get("/api/v1/wallets");
      setWallets(response.data.wallets || []);
    } catch (err: any) {
      console.error("Failed to load wallets:", err);
      setError("Failed to load wallets");
      // Keep wallets empty if API fails
      setWallets([]);
    } finally {
      setLoading(false);
    }
  };

  const loadTransfers = async () => {
    try {
      const response = await api.get("/api/v1/transfers");
      setTransfers(response.data.transfers || []);
    } catch (err: any) {
      console.error("Failed to load transfers:", err);
      // Keep transfers empty if API fails
      setTransfers([]);
    }
  };

  const createWallet = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      setLoading(true);
      setError(null);

      const response = await api.post("/api/v1/wallets", {
        name: createWalletForm.name,
        type: createWalletForm.type,
        coin: createWalletForm.coin,
        passphrase: createWalletForm.passphrase,
      });

      // Add new wallet to the list
      setWallets((prev) => [...prev, response.data]);

      // Reset form and go back to wallets tab
      setCreateWalletForm({
        name: "",
        type: "warm",
        coin: "BTC",
        passphrase: "",
      });
      setActiveTab("wallets");
    } catch (err: any) {
      setError(
        err.response?.data?.message || err.message || "Failed to create wallet"
      );
    } finally {
      setLoading(false);
    }
  };

  const getStatusColor = (status: string) => {
    const colors = {
      draft: "bg-gray-100 text-gray-800",
      submitted: "bg-blue-100 text-blue-800",
      pending_approval: "bg-yellow-100 text-yellow-800",
      signed: "bg-purple-100 text-purple-800",
      broadcast: "bg-indigo-100 text-indigo-800",
      confirmed: "bg-green-100 text-green-800",
      failed: "bg-red-100 text-red-800",
    };
    return colors[status as keyof typeof colors] || "bg-gray-100 text-gray-800";
  };

  const getWalletTypeColor = (type: string) => {
    const colors = {
      hot: "bg-red-100 text-red-800",
      warm: "bg-orange-100 text-orange-800",
      cold: "bg-blue-100 text-blue-800",
    };
    return colors[type as keyof typeof colors] || "bg-gray-100 text-gray-800";
  };

  const renderOverview = () => (
    <div className="grid gap-8">
      {/* Stats Grid */}
      <div className="grid md:grid-cols-4 gap-6">
        <div className="bg-white rounded-lg shadow p-6">
          <div className="flex items-center">
            <div className="w-12 h-12 bg-blue-100 rounded-lg flex items-center justify-center">
              <svg
                className="w-6 h-6 text-blue-600"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"
                />
              </svg>
            </div>
            <div className="ml-4">
              <p className="text-sm font-medium text-gray-600">Total Wallets</p>
              <p className="text-2xl font-semibold text-gray-900">
                {wallets.length}
              </p>
            </div>
          </div>
        </div>

        <div className="bg-white rounded-lg shadow p-6">
          <div className="flex items-center">
            <div className="w-12 h-12 bg-yellow-100 rounded-lg flex items-center justify-center">
              <svg
                className="w-6 h-6 text-yellow-600"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
                />
              </svg>
            </div>
            <div className="ml-4">
              <p className="text-sm font-medium text-gray-600">
                Pending Transfers
              </p>
              <p className="text-2xl font-semibold text-gray-900">
                {
                  transfers.filter((t) =>
                    ["submitted", "pending_approval"].includes(t.status)
                  ).length
                }
              </p>
            </div>
          </div>
        </div>

        <div className="bg-white rounded-lg shadow p-6">
          <div className="flex items-center">
            <div className="w-12 h-12 bg-green-100 rounded-lg flex items-center justify-center">
              <svg
                className="w-6 h-6 text-green-600"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
                />
              </svg>
            </div>
            <div className="ml-4">
              <p className="text-sm font-medium text-gray-600">
                Completed Today
              </p>
              <p className="text-2xl font-semibold text-gray-900">
                {transfers.filter((t) => t.status === "confirmed").length}
              </p>
            </div>
          </div>
        </div>

        <div className="bg-white rounded-lg shadow p-6">
          <div className="flex items-center">
            <div className="w-12 h-12 bg-purple-100 rounded-lg flex items-center justify-center">
              <svg
                className="w-6 h-6 text-purple-600"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M13 10V3L4 14h7v7l9-11h-7z"
                />
              </svg>
            </div>
            <div className="ml-4">
              <p className="text-sm font-medium text-gray-600">
                Warm Transfers
              </p>
              <p className="text-2xl font-semibold text-gray-900">
                {transfers.filter((t) => t.type === "warm").length}
              </p>
            </div>
          </div>
        </div>
      </div>

      {/* Quick Actions */}
      <div className="bg-white rounded-lg shadow p-6">
        <h3 className="text-lg font-semibold text-gray-900 mb-4">
          Quick Actions
        </h3>
        <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-4">
          <button
            onClick={() => setActiveTab("new-transfer")}
            className="flex items-center justify-center gap-3 p-4 border-2 border-dashed border-blue-300 rounded-lg hover:border-blue-500 hover:bg-blue-50 transition-colors"
          >
            <svg
              className="w-6 h-6 text-blue-600"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M12 4v16m8-8H4"
              />
            </svg>
            <span className="font-medium text-blue-600">New Transfer</span>
          </button>

          <button
            onClick={() => setActiveTab("wallets")}
            className="flex items-center justify-center gap-3 p-4 border-2 border-dashed border-green-300 rounded-lg hover:border-green-500 hover:bg-green-50 transition-colors"
          >
            <svg
              className="w-6 h-6 text-green-600"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"
              />
            </svg>
            <span className="font-medium text-green-600">View Wallets</span>
          </button>

          <button
            onClick={() => setActiveTab("transfers")}
            className="flex items-center justify-center gap-3 p-4 border-2 border-dashed border-purple-300 rounded-lg hover:border-purple-500 hover:bg-purple-50 transition-colors"
          >
            <svg
              className="w-6 h-6 text-purple-600"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M9 5H7a2 2 0 00-2 2v10a2 2 0 002 2h8a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"
              />
            </svg>
            <span className="font-medium text-purple-600">
              Transfer History
            </span>
          </button>

          <button className="flex items-center justify-center gap-3 p-4 border-2 border-dashed border-gray-300 rounded-lg hover:border-gray-500 hover:bg-gray-50 transition-colors">
            <svg
              className="w-6 h-6 text-gray-600"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v4"
              />
            </svg>
            <span className="font-medium text-gray-600">Reports</span>
          </button>
        </div>
      </div>

      {/* Recent Activity */}
      <div className="bg-white rounded-lg shadow">
        <div className="px-6 py-4 border-b border-gray-200">
          <h3 className="text-lg font-semibold text-gray-900">
            Recent Activity
          </h3>
        </div>
        <div className="divide-y divide-gray-200">
          {transfers.slice(0, 5).map((transfer) => (
            <div key={transfer.id} className="px-6 py-4">
              <div className="flex items-center justify-between">
                <div className="flex items-center space-x-3">
                  <div
                    className={`w-2 h-2 rounded-full ${
                      transfer.status === "confirmed"
                        ? "bg-green-400"
                        : transfer.status === "pending_approval"
                        ? "bg-yellow-400"
                        : "bg-blue-400"
                    }`}
                  ></div>
                  <div>
                    <p className="text-sm font-medium text-gray-900">
                      {transfer.type === "warm" ? "Warm" : "Cold"} transfer{" "}
                      {transfer.status.replace("_", " ")}
                    </p>
                    <p className="text-sm text-gray-500">
                      {transfer.amount} {transfer.coin} to{" "}
                      {transfer.destination.substring(0, 8)}...
                    </p>
                  </div>
                </div>
                <div className="text-right">
                  <span
                    className={`inline-flex px-2 py-1 text-xs font-semibold rounded-full ${getStatusColor(
                      transfer.status
                    )}`}
                  >
                    {transfer.status.replace("_", " ")}
                  </span>
                  <p className="text-sm text-gray-500 mt-1">
                    {new Date(transfer.createdAt).toLocaleDateString()}
                  </p>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
  const renderWallets = () => (
    <div className="space-y-6">
      <div className="bg-white rounded-lg shadow">
        <div className="px-6 py-4 border-b border-gray-200">
          <div className="flex items-center justify-between">
            <h3 className="text-lg font-semibold text-gray-900">Wallets</h3>
            <button
              onClick={() => setActiveTab("create-wallet")}
              className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg text-sm font-semibold transition-colors"
            >
              Add Wallet
            </button>
          </div>
        </div>
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Wallet
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Type
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Coin
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Balance
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Status
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {wallets.map((wallet) => (
                <tr key={wallet.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="font-medium text-gray-900">
                      {wallet.name}
                    </div>
                    <div className="text-sm text-gray-500">ID: {wallet.id}</div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span
                      className={`inline-flex px-2 py-1 text-xs font-semibold rounded-full ${getWalletTypeColor(
                        wallet.type
                      )}`}
                    >
                      {wallet.type}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                    {wallet.coin}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                    {wallet.balance}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span
                      className={`inline-flex px-2 py-1 text-xs font-semibold rounded-full ${
                        wallet.status === "active"
                          ? "bg-green-100 text-green-800"
                          : "bg-gray-100 text-gray-800"
                      }`}
                    >
                      {wallet.status}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    <button className="text-blue-600 hover:text-blue-900 mr-4">
                      View
                    </button>
                    <button className="text-green-600 hover:text-green-900">
                      Transfer
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );

  const renderTransfers = () => (
    <div className="space-y-6">
      <div className="bg-white rounded-lg shadow">
        <div className="px-6 py-4 border-b border-gray-200">
          <div className="flex items-center justify-between">
            <h3 className="text-lg font-semibold text-gray-900">
              Transfer History
            </h3>
            <div className="flex gap-2">
              <select className="border border-gray-300 rounded-md px-3 py-2 text-sm text-gray-900">
                <option value="">All Types</option>
                <option value="warm">Warm</option>
                <option value="cold">Cold</option>
              </select>
              <select className="border border-gray-300 rounded-md px-3 py-2 text-sm text-gray-900">
                <option value="">All Status</option>
                <option value="pending_approval">Pending Approval</option>
                <option value="confirmed">Confirmed</option>
                <option value="failed">Failed</option>
              </select>
            </div>
          </div>
        </div>
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Transfer
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Type
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Amount
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Destination
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Status
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Date
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {transfers.map((transfer) => (
                <tr key={transfer.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="font-medium text-gray-900">
                      {transfer.walletName}
                    </div>
                    <div className="text-sm text-gray-500">
                      ID: {transfer.id}
                    </div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span
                      className={`inline-flex px-2 py-1 text-xs font-semibold rounded-full ${
                        transfer.type === "warm"
                          ? "bg-orange-100 text-orange-800"
                          : "bg-blue-100 text-blue-800"
                      }`}
                    >
                      {transfer.type}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                    {transfer.amount} {transfer.coin}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                    {transfer.destination.substring(0, 12)}...
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span
                      className={`inline-flex px-2 py-1 text-xs font-semibold rounded-full ${getStatusColor(
                        transfer.status
                      )}`}
                    >
                      {transfer.status.replace("_", " ")}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {new Date(transfer.createdAt).toLocaleDateString()}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    <button className="text-blue-600 hover:text-blue-900 mr-4">
                      View Details
                    </button>
                    {transfer.status === "pending_approval" && (
                      <button className="text-green-600 hover:text-green-900">
                        Approve
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );

  const renderCreateWallet = () => (
    <div className="space-y-6">
      <div className="bg-white rounded-lg shadow p-6">
        <h3 className="text-lg font-semibold text-gray-900 mb-6">
          Create New Wallet
        </h3>

        {error && (
          <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg">
            <p className="text-sm text-red-700">{error}</p>
          </div>
        )}

        <form onSubmit={createWallet} className="space-y-6">
          {/* Wallet Name */}
          <div>
            <label
              htmlFor="walletName"
              className="block text-sm font-medium text-gray-700 mb-2"
            >
              Wallet Name
            </label>
            <input
              type="text"
              id="walletName"
              value={createWalletForm.name}
              onChange={(e) =>
                setCreateWalletForm((prev) => ({
                  ...prev,
                  name: e.target.value,
                }))
              }
              required
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-gray-900 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              placeholder="e.g., Main Trading Wallet"
            />
          </div>

          {/* Wallet Type */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-3">
              Wallet Type
            </label>
            <div className="grid grid-cols-2 gap-4">
              <button
                type="button"
                onClick={() =>
                  setCreateWalletForm((prev) => ({ ...prev, type: "warm" }))
                }
                className={`p-4 border-2 rounded-lg text-left transition-colors ${
                  createWalletForm.type === "warm"
                    ? "border-orange-500 bg-orange-50"
                    : "border-gray-300 hover:border-gray-400"
                }`}
              >
                <div className="flex items-center gap-3">
                  <div className="w-3 h-3 bg-orange-500 rounded-full"></div>
                  <div>
                    <h4 className="font-semibold text-gray-900">Warm Wallet</h4>
                    <p className="text-sm text-gray-600">
                      For frequent trading and transfers
                    </p>
                  </div>
                </div>
              </button>
              <button
                type="button"
                onClick={() =>
                  setCreateWalletForm((prev) => ({ ...prev, type: "cold" }))
                }
                className={`p-4 border-2 rounded-lg text-left transition-colors ${
                  createWalletForm.type === "cold"
                    ? "border-blue-500 bg-blue-50"
                    : "border-gray-300 hover:border-gray-400"
                }`}
              >
                <div className="flex items-center gap-3">
                  <div className="w-3 h-3 bg-blue-500 rounded-full"></div>
                  <div>
                    <h4 className="font-semibold text-gray-900">Cold Wallet</h4>
                    <p className="text-sm text-gray-600">
                      For long-term secure storage
                    </p>
                  </div>
                </div>
              </button>
            </div>
          </div>

          {/* Cryptocurrency */}
          <div>
            <label
              htmlFor="coin"
              className="block text-sm font-medium text-gray-700 mb-2"
            >
              Cryptocurrency
            </label>
            <select
              id="coin"
              value={createWalletForm.coin}
              onChange={(e) =>
                setCreateWalletForm((prev) => ({
                  ...prev,
                  coin: e.target.value,
                }))
              }
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-gray-900 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            >
              <option value="BTC">Bitcoin (BTC)</option>
              <option value="ETH">Ethereum (ETH)</option>
              <option value="LTC">Litecoin (LTC)</option>
              <option value="BCH">Bitcoin Cash (BCH)</option>
            </select>
          </div>

          {/* Passphrase */}
          <div>
            <label
              htmlFor="passphrase"
              className="block text-sm font-medium text-gray-700 mb-2"
            >
              Wallet Passphrase
            </label>
            <input
              type="password"
              id="passphrase"
              value={createWalletForm.passphrase}
              onChange={(e) =>
                setCreateWalletForm((prev) => ({
                  ...prev,
                  passphrase: e.target.value,
                }))
              }
              required
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-gray-900 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              placeholder="Enter a strong passphrase"
            />
            <p className="text-sm text-gray-500 mt-1">
              This passphrase will be used to secure your wallet. Make sure to
              store it safely.
            </p>
          </div>

          {/* Action Buttons */}
          <div className="flex gap-4 pt-6">
            <button
              type="submit"
              disabled={loading}
              className="flex-1 bg-blue-600 hover:bg-blue-700 disabled:bg-blue-400 text-white py-3 px-6 rounded-lg font-semibold transition-colors flex items-center justify-center gap-2"
            >
              {loading ? (
                <>
                  <svg
                    className="animate-spin w-5 h-5"
                    fill="none"
                    viewBox="0 0 24 24"
                  >
                    <circle
                      className="opacity-25"
                      cx="12"
                      cy="12"
                      r="10"
                      stroke="currentColor"
                      strokeWidth="4"
                    ></circle>
                    <path
                      className="opacity-75"
                      fill="currentColor"
                      d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                    ></path>
                  </svg>
                  Creating Wallet...
                </>
              ) : (
                <>Create Wallet</>
              )}
            </button>
            <button
              type="button"
              onClick={() => setActiveTab("wallets")}
              className="px-6 py-3 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors"
            >
              Cancel
            </button>
          </div>
        </form>

        {/* Security Notice */}
        <div className="mt-6 p-4 bg-yellow-50 border border-yellow-200 rounded-lg">
          <div className="flex items-start gap-3">
            <svg
              className="w-5 h-5 text-yellow-600 mt-0.5"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.732 16c-.77.833.192 2.5 1.732 2.5z"
              />
            </svg>
            <div>
              <h4 className="font-semibold text-yellow-800">Security Notice</h4>
              <p className="text-sm text-yellow-700 mt-1">
                Your wallet will be created with multi-signature security
                powered by BitGo. The passphrase you provide will be used to
                encrypt your wallet keys.
                {createWalletForm.type === "cold" &&
                  " Cold wallets require additional offline verification steps."}
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );

  const renderNewTransfer = () => (
    <div className="space-y-6">
      \n{" "}
      <div className="bg-white rounded-lg shadow p-6">
        <h3 className="text-lg font-semibold text-gray-900 mb-6">
          Create New Transfer
        </h3>

        {/* Wallet Type Selection */}
        <div className="mb-6">
          <label className="block text-sm font-medium text-gray-700 mb-3">
            Transfer Type
          </label>
          <div className="grid grid-cols-2 gap-4">
            <button
              onClick={() => setSelectedWalletType("warm")}
              className={`p-4 border-2 rounded-lg text-left transition-colors ${
                selectedWalletType === "warm"
                  ? "border-orange-500 bg-orange-50"
                  : "border-gray-300 hover:border-gray-400"
              }`}
            >
              <div className="flex items-center gap-3">
                <div className="w-3 h-3 bg-orange-500 rounded-full"></div>
                <div>
                  <h4 className="font-semibold text-gray-900">Warm Transfer</h4>
                  <p className="text-sm text-gray-600">
                    Fast processing with approval workflow
                  </p>
                </div>
              </div>
            </button>
            <button
              onClick={() => setSelectedWalletType("cold")}
              className={`p-4 border-2 rounded-lg text-left transition-colors ${
                selectedWalletType === "cold"
                  ? "border-blue-500 bg-blue-50"
                  : "border-gray-300 hover:border-gray-400"
              }`}
            >
              <div className="flex items-center gap-3">
                <div className="w-3 h-3 bg-blue-500 rounded-full"></div>
                <div>
                  <h4 className="font-semibold text-gray-900">Cold Transfer</h4>
                  <p className="text-sm text-gray-600">
                    Secure offline processing with enhanced security
                  </p>
                </div>
              </div>
            </button>
          </div>
        </div>

        <form className="space-y-6">
          {/* Source Wallet */}
          <div>
            <label
              htmlFor="sourceWallet"
              className="block text-sm font-medium text-gray-700 mb-2"
            >
              Source Wallet
            </label>
            <select
              id="sourceWallet"
              className="w-full border border-gray-300 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            >
              <option value="">Select a wallet</option>
              {wallets
                .filter((w) => w.type === selectedWalletType)
                .map((wallet) => (
                  <option key={wallet.id} value={wallet.id}>
                    {wallet.name} ({wallet.balance} {wallet.coin})
                  </option>
                ))}
            </select>
          </div>

          {/* Destination Address */}
          <div>
            <label
              htmlFor="destination"
              className="block text-sm font-medium text-gray-700 mb-2"
            >
              Destination Address
            </label>
            <input
              type="text"
              id="destination"
              className="w-full border border-gray-300 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              placeholder="Enter destination address"
            />
          </div>

          {/* Amount */}
          <div>
            <label
              htmlFor="amount"
              className="block text-sm font-medium text-gray-700 mb-2"
            >
              Amount
            </label>
            <div className="relative">
              <input
                type="number"
                id="amount"
                step="0.00000001"
                className="w-full border border-gray-300 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                placeholder="0.00000000"
              />
              <div className="absolute inset-y-0 right-0 flex items-center pr-3">
                <span className="text-gray-500 text-sm">BTC</span>
              </div>
            </div>
          </div>

          {/* Cold Transfer Additional Fields */}
          {selectedWalletType === "cold" && (
            <>
              <div>
                <label
                  htmlFor="businessPurpose"
                  className="block text-sm font-medium text-gray-700 mb-2"
                >
                  Business Purpose
                </label>
                <textarea
                  id="businessPurpose"
                  rows={3}
                  className="w-full border border-gray-300 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  placeholder="Explain the business purpose of this transfer"
                />
              </div>

              <div>
                <label
                  htmlFor="urgency"
                  className="block text-sm font-medium text-gray-700 mb-2"
                >
                  Urgency Level
                </label>
                <select
                  id="urgency"
                  className="w-full border border-gray-300 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                >
                  <option value="standard">Standard (3-5 business days)</option>
                  <option value="urgent">Urgent (1-2 business days)</option>
                  <option value="emergency">Emergency (Same day)</option>
                </select>
              </div>
            </>
          )}

          {/* Action Buttons */}
          <div className="flex gap-4 pt-6">
            <button
              type="submit"
              className="flex-1 bg-blue-600 hover:bg-blue-700 text-white py-3 px-6 rounded-lg font-semibold transition-colors"
            >
              {selectedWalletType === "warm"
                ? "Submit Transfer"
                : "Submit Cold Transfer Request"}
            </button>
            <button
              type="button"
              onClick={() => setActiveTab("overview")}
              className="px-6 py-3 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors"
            >
              Cancel
            </button>
          </div>
        </form>

        {selectedWalletType === "cold" && (
          <div className="mt-6 p-4 bg-yellow-50 border border-yellow-200 rounded-lg">
            <div className="flex items-start gap-3">
              <svg
                className="w-5 h-5 text-yellow-600 mt-0.5"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.732 16c-.77.833.192 2.5 1.732 2.5z"
                />
              </svg>
              <div>
                <h4 className="font-semibold text-yellow-800">
                  Cold Transfer Processing
                </h4>
                <p className="text-sm text-yellow-700 mt-1">
                  Cold transfers require offline processing and additional
                  security validations. Processing times vary based on urgency
                  level and may require manual approvals.
                </p>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <header className="bg-white shadow-sm border-b border-gray-200">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            <div className="flex items-center">
              <h1 className="text-xl font-semibold text-gray-900">
                BitGo Wallets - Custody Platform
              </h1>
            </div>
            <button
              onClick={onLogout}
              className="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-gray-700 hover:text-gray-900 hover:bg-gray-100 rounded-lg transition-colors"
            >
              <svg
                className="w-4 h-4"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1"
                />
              </svg>
              Logout
            </button>
          </div>
        </div>
      </header>

      {/* Navigation Tabs */}
      <div className="bg-white border-b border-gray-200">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <nav className="flex space-x-8">
            {[
              {
                key: "overview",
                label: "Overview",
                icon: "M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2H5a2 2 0 00-2-2z",
              },
              {
                key: "wallets",
                label: "Wallets",
                icon: "M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10",
              },
              {
                key: "transfers",
                label: "Transfers",
                icon: "M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4",
              },
              {
                key: "new-transfer",
                label: "New Transfer",
                icon: "M12 4v16m8-8H4",
              },
            ].map((tab) => (
              <button
                key={tab.key}
                onClick={() => setActiveTab(tab.key as any)}
                className={`flex items-center gap-2 py-4 px-1 border-b-2 font-medium text-sm ${
                  activeTab === tab.key
                    ? "border-blue-500 text-blue-600"
                    : "border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300"
                }`}
              >
                <svg
                  className="w-5 h-5"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d={tab.icon}
                  />
                </svg>
                {tab.label}
              </button>
            ))}
          </nav>
        </div>
      </div>

      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {activeTab === "overview" && renderOverview()}
        {activeTab === "wallets" && renderWallets()}
        {activeTab === "transfers" && renderTransfers()}
        {activeTab === "new-transfer" && renderNewTransfer()}
        {activeTab === "create-wallet" && renderCreateWallet()}
      </main>
    </div>
  );
};
