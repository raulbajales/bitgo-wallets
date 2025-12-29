"use client";

import { useState, useEffect } from "react";
import { LandingPage } from "@/components/LandingPage";
import { LoginPage } from "@/components/auth/LoginPage";
import { Dashboard } from "@/components/Dashboard";

export default function Home() {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [showLogin, setShowLogin] = useState(false);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    // Check if user is already logged in
    const token = localStorage.getItem("auth_token");
    setIsAuthenticated(!!token);
    setIsLoading(false);
  }, []);

  const handleLoginSuccess = (token: string) => {
    setIsAuthenticated(true);
    setShowLogin(false);
  };

  const handleLogout = () => {
    localStorage.removeItem("auth_token");
    setIsAuthenticated(false);
    setShowLogin(false);
  };

  const handleLoginClick = () => {
    setShowLogin(true);
  };

  const handleLoginCancel = () => {
    setShowLogin(false);
  };

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-xl">Loading...</div>
      </div>
    );
  }

  if (isAuthenticated) {
    return <Dashboard onLogout={handleLogout} />;
  }

  if (showLogin) {
    return (
      <LoginPage
        onLoginSuccess={handleLoginSuccess}
        onCancel={handleLoginCancel}
      />
    );
  }

  return <LandingPage onLoginClick={handleLoginClick} />;
}
