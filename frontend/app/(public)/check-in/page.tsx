"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { PlaneTakeoff } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";

export default function CheckInHomePage() {
  const router = useRouter();
  const [pnr, setPnr] = useState("");
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!pnr.trim()) {
      setError("Please enter your booking reference (PNR).");
      return;
    }
    setError(null);
    router.push(`/check-in/${pnr.trim().toUpperCase()}`);
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50 px-4 dark:bg-gray-950">
      <Card className="w-full max-w-md">
        <CardHeader>
          <div className="flex items-center gap-2">
            <PlaneTakeoff className="h-5 w-5 text-brand-indigo" />
            <CardTitle className="text-2xl">Online Check-in</CardTitle>
          </div>
          <CardDescription>Enter your booking reference to check in</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            {error && <div className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-900 dark:bg-red-950 dark:text-red-400">{error}</div>}
            <div>
              <label className="mb-1 block text-sm font-medium text-gray-600 dark:text-gray-400">Booking Reference (PNR)</label>
              <input
                type="text"
                required
                maxLength={6}
                value={pnr}
                onChange={e => setPnr(e.target.value.toUpperCase())}
                placeholder="A7KF3L"
                className="flex h-12 w-full rounded-md border border-input bg-background px-3 py-2 text-lg font-semibold tracking-widest"
              />
            </div>
            <Button type="submit" className="btn-gradient w-full">Check In</Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
