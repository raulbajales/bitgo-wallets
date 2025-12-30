"use client";

import { useState } from "react";
import { LandingPage } from "@/components/LandingPage";
import { Dashboard } from "@/components/Dashboard";

export default function Home() {
  const [showDashboard, setShowDashboard] = useState(false);

  const handleGetStarted = () => {
    setShowDashboard(true);
  };

  if (showDashboard) {
    return <Dashboard />;
  }

  return <LandingPage onGetStarted={handleGetStarted} />;
}
