"use client";

import React, { useState, useEffect } from "react";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { WalletCard, type Wallet } from "./wallet-card";
import {
  CreateTransferForm,
  type TransferFormData,
} from "../transfers/create-transfer-form";
import {
  CreateWalletForm,
  type CreateWalletFormData,
} from "./create-wallet-form";
import api from "@/lib/api";
export function WalletDashboard() {
  const [wallets, setWallets] = useState<Wallet[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [syncing, setSyncing] = useState<string | null>(null);
  const [showTransferForm, setShowTransferForm] = useState(false);
  const [selectedWallet, setSelectedWallet] = useState<Wallet | null>(null);
  const [showCreateWalletForm, setShowCreateWalletForm] = useState(false);

  // Load wallets from API
  useEffect(() => {
    loadWallets();
  }, []);

  const loadWallets = async () => {
    try {
      setLoading(true);
      setError(null);

      const response = await api.get("/api/v1/wallets");
      setWallets(response.data.wallets || []);
    } catch (err) {
      console.error("Failed to load wallets:", err);
      setError("Failed to load wallets");
    } finally {
      setLoading(false);
    }
  };

  const handleDiscoverWallets = async () => {
    try {
      setLoading(true);
      setError(null);

      const response = await api.get("/api/v1/wallets/discover");

      // Reload wallets to get the newly discovered ones
      await loadWallets();
    } catch (err) {
      console.error("Failed to discover wallets:", err);
      setError("Failed to discover wallets");
      setLoading(false);
    }
  };

  const handleViewDetails = (wallet: Wallet) => {
    // TODO: Navigate to wallet detail page
    console.log("View details for wallet:", wallet.id);
  };

  const handleCreateTransfer = (wallet: Wallet) => {
    setSelectedWallet(wallet);
    setShowTransferForm(true);
  };

  const handleTransferSubmit = async (transferData: TransferFormData) => {
    if (!selectedWallet) {
      throw new Error("No wallet selected");
    }

    try {
      setError(null);

      // Call the API to create the transfer
      const response = await api.post(
        `/api/v1/wallets/${selectedWallet.id}/transfers`,
        {
          recipient_address: transferData.recipientAddress,
          amount_string: transferData.amountString,
          coin: transferData.coin,
          transfer_type: transferData.transferType,
          memo: transferData.memo,
          business_purpose: transferData.businessPurpose,
          requestor_name: transferData.requestorName,
          requestor_email: transferData.requestorEmail,
          urgency_level: transferData.urgencyLevel,
          auto_process: transferData.autoProcess,
        }
      );

      // Close the form and reload wallets
      setShowTransferForm(false);
      setSelectedWallet(null);
      await loadWallets();

      // Could show a success message here
    } catch (err) {
      // Let the form handle the error display
      throw err;
    }
  };

  const handleCancelTransfer = () => {
    setShowTransferForm(false);
    setSelectedWallet(null);
  };

  const handleCreateWallet = () => {
    setShowCreateWalletForm(true);
  };

  const handleWalletSubmit = async (walletData: CreateWalletFormData) => {
    try {
      setError(null);

      // Call the API to create the wallet
      const response = await api.post("/api/v1/wallets", {
        bitgo_wallet_id: walletData.bitgoWalletId,
        label: walletData.label,
        coin: walletData.coin,
        wallet_type: walletData.walletType,
        multisig_type: walletData.multisigType,
        threshold: walletData.threshold,
        tags: walletData.tags,
        metadata: walletData.metadata,
      });

      // Close the form and reload wallets
      setShowCreateWalletForm(false);
      await loadWallets();

      // Could show a success message here
    } catch (err) {
      // Let the form handle the error display
      throw err;
    }
  };

  const handleCancelWalletCreation = () => {
    setShowCreateWalletForm(false);
  };

  const handleSyncBalance = async (wallet: Wallet) => {
    try {
      setSyncing(wallet.id);
      setError(null);

      await api.post(`/api/v1/wallets/${wallet.id}/sync-balance`);

      // Reload wallets to get updated balances
      await loadWallets();
      setSyncing(null);
    } catch (err) {
      console.error("Failed to sync wallet balance:", err);
      setError("Failed to sync wallet balance");
      setSyncing(null);
    }
  };

  const getTotalBalance = () => {
    return wallets.reduce((total, wallet) => {
      // Simple sum - in reality you'd need exchange rates
      const balance = parseFloat(wallet.balanceString);
      return total + (isNaN(balance) ? 0 : balance);
    }, 0);
  };

  const getWalletTypeCount = (type: string) => {
    return wallets.filter((w) => w.walletType === type).length;
  };

  if (loading && wallets.length === 0) {
    return (
      <div className="min-h-screen bg-gray-50 p-6">
        <div className="max-w-7xl mx-auto">
          <div className="text-center py-12">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto mb-4"></div>
            <p className="text-gray-600">Loading wallets...</p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50 p-6">
      <div className="max-w-7xl mx-auto">
        {/* Header */}
        <div className="mb-8">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-3xl font-bold text-gray-900">
                Wallet Dashboard
              </h1>
              <p className="text-gray-600 mt-1">
                Manage your BitGo wallets and transfers
              </p>
            </div>
            <div className="flex space-x-3">
              <Button
                variant="outline"
                onClick={handleDiscoverWallets}
                disabled={loading}
              >
                {loading ? "Discovering..." : "Discover Wallets"}
              </Button>
              <Button onClick={handleCreateWallet}>Create Wallet</Button>
            </div>
          </div>
        </div>

        {/* Summary Cards */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-gray-500">
                Total Wallets
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-3xl font-bold">{wallets.length}</div>
              <p className="text-sm text-gray-500 mt-1">
                {wallets.filter((w) => w.isActive).length} active
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-gray-500">
                Warm Wallets
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-3xl font-bold">
                {getWalletTypeCount("custodial")}
              </div>
              <Badge variant="default" className="mt-2">
                Ready to use
              </Badge>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-gray-500">
                Cold Wallets
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-3xl font-bold">
                {getWalletTypeCount("cold")}
              </div>
              <Badge variant="secondary" className="mt-2">
                High security
              </Badge>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-gray-500">
                Hot Wallets
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-3xl font-bold">
                {getWalletTypeCount("hot")}
              </div>
              <Badge variant="warning" className="mt-2">
                Operations
              </Badge>
            </CardContent>
          </Card>
        </div>

        {/* Error Display */}
        {error && (
          <Card className="mb-6 border-red-200 bg-red-50">
            <CardContent className="pt-6">
              <div className="flex items-center space-x-2">
                <span className="text-red-600">⚠️</span>
                <p className="text-red-700">{error}</p>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setError(null)}
                  className="ml-auto"
                >
                  Dismiss
                </Button>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Wallets Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {wallets.map((wallet) => (
            <WalletCard
              key={wallet.id}
              wallet={wallet}
              onViewDetails={handleViewDetails}
              onCreateTransfer={handleCreateTransfer}
              onSyncBalance={handleSyncBalance}
            />
          ))}
        </div>

        {/* Empty State */}
        {wallets.length === 0 && !loading && (
          <Card className="text-center py-12">
            <CardContent>
              <div className="text-gray-400 mb-4">
                <svg
                  className="mx-auto h-16 w-16"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={1}
                    d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"
                  />
                </svg>
              </div>
              <h3 className="text-xl font-medium text-gray-900 mb-2">
                No wallets found
              </h3>
              <p className="text-gray-500 mb-6">
                Discover wallets from BitGo or create a new wallet to get
                started.
              </p>
              <div className="flex justify-center space-x-4">
                <Button variant="outline" onClick={handleDiscoverWallets}>
                  Discover Wallets
                </Button>
                <Button onClick={handleCreateWallet}>Create Wallet</Button>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Loading Overlay for Sync */}
        {syncing && (
          <div className="fixed inset-0 bg-black bg-opacity-25 flex items-center justify-center z-50">
            <Card className="p-6">
              <div className="flex items-center space-x-3">
                <div className="animate-spin rounded-full h-5 w-5 border-b-2 border-blue-600"></div>
                <p>Syncing wallet balance...</p>
              </div>
            </Card>
          </div>
        )}

        {/* Transfer Form Modal */}
        {showTransferForm && selectedWallet && (
          <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
            <div className="bg-white rounded-lg max-w-4xl w-full max-h-[90vh] overflow-y-auto">
              <CreateTransferForm
                wallet={selectedWallet}
                onSubmit={handleTransferSubmit}
                onCancel={handleCancelTransfer}
              />
            </div>
          </div>
        )}

        {/* Wallet Creation Form Modal */}
        {showCreateWalletForm && (
          <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
            <div className="bg-white rounded-lg max-w-4xl w-full max-h-[90vh] overflow-y-auto">
              <CreateWalletForm
                onSubmit={handleWalletSubmit}
                onCancel={handleCancelWalletCreation}
              />
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
