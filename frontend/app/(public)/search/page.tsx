"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { PlaneTakeoff, Loader2, SearchX } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { flightApi, type SearchResult } from "@/lib/api-client";

export default function SearchPage() {
  const router = useRouter();
  const [origin, setOrigin] = useState("");
  const [destination, setDestination] = useState("");
  const [date, setDate] = useState("");
  const [passengers, setPassengers] = useState(1);
  const [results, setResults] = useState<SearchResult[] | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSearch = async () => {
    if (!origin || !destination || !date) return;
    setLoading(true);
    setError(null);
    try {
      const res = await flightApi.search({
        origin,
        destination,
        date,
        passengers: String(passengers),
      });
      setResults(res.data);
    } catch (e) {
      setResults(null);
      setError(e instanceof Error ? e.message : "Search failed");
    } finally {
      setLoading(false);
    }
  };

  const select = (id: number) => {
    router.push(`/flights/${id}?passengers=${passengers}`);
  };

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950">
      <div className="bg-brand-gradient pb-8 pt-6">
        <div className="container mx-auto px-4">
          <Card className="border-0 shadow-2xl">
            <CardContent className="p-4">
              <div className="flex flex-wrap items-end gap-3">
                <div className="flex-1 min-w-[200px]">
                  <label className="mb-1 block text-xs font-medium text-gray-500">From</label>
                  <input type="text" value={origin} onChange={e => setOrigin(e.target.value.toUpperCase())} placeholder="JFK" className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm" />
                </div>
                <button type="button" className="mb-0.5 rounded-full border p-2 text-gray-500 hover:bg-gray-100" onClick={() => { setOrigin(destination); setDestination(origin); }} aria-label="Swap airports"><PlaneTakeoff className="h-4 w-4 rotate-90" /></button>
                <div className="flex-1 min-w-[200px]">
                  <label className="mb-1 block text-xs font-medium text-gray-500">To</label>
                  <input type="text" value={destination} onChange={e => setDestination(e.target.value.toUpperCase())} placeholder="LAX" className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm" />
                </div>
                <div className="w-[180px]">
                  <label className="mb-1 block text-xs font-medium text-gray-500">Date</label>
                  <input type="date" value={date} onChange={e => setDate(e.target.value)} className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm" />
                </div>
                <div className="w-[100px]">
                  <label className="mb-1 block text-xs font-medium text-gray-500">Passengers</label>
                  <select value={passengers} onChange={e => setPassengers(parseInt(e.target.value))} className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm">
                    {[1,2,3,4,5,6].map(n => <option key={n} value={n}>{n} {n === 1 ? "Adult" : "Adults"}</option>)}
                  </select>
                </div>
                <button className="btn-gradient h-10 px-8 rounded-md text-sm font-semibold text-white disabled:opacity-50" onClick={handleSearch} disabled={loading}>
                  {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : "Search"}
                </button>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>

      <div className="container mx-auto px-4 py-8">
        {error && (
          <div className="mb-6 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-900 dark:bg-red-950 dark:text-red-400">
            {error}
          </div>
        )}

        {loading && (
          <div className="flex items-center justify-center gap-2 py-16 text-gray-500">
            <Loader2 className="h-5 w-5 animate-spin" /> Searching available flights…
          </div>
        )}

        {!loading && !error && results !== null && results.length === 0 && (
          <div className="rounded-lg border bg-white p-8 text-center shadow-sm dark:border-gray-700 dark:bg-gray-900">
            <SearchX className="mx-auto mb-4 h-12 w-12 text-gray-300 dark:text-gray-600" />
            <h3 className="mb-2 text-lg font-semibold">No flights found</h3>
            <p className="text-sm text-gray-500 dark:text-gray-400">Try different airports or a different date.</p>
          </div>
        )}

        {!loading && results !== null && results.length > 0 && (
          <div className="space-y-4">
            <p className="text-sm text-gray-500 dark:text-gray-400">{results.length} flight{results.length > 1 ? "s" : ""} found</p>
            {results.map(f => {
              const cheapest = [...f.fares].sort((a, b) => a.total - b.total)[0];
              return (
                <Card key={f.flight_id} className="hover:shadow-md transition-shadow">
                  <CardContent className="p-4 flex items-center justify-between flex-wrap gap-4">
                    <div className="flex items-center gap-4">
                      <div>
                        <div className="text-lg font-bold">{f.flight_number}</div>
                        <div className="text-xs text-gray-500">{f.airline.name} · {f.aircraft}</div>
                      </div>
                    </div>
                    <div className="flex items-center gap-4 text-sm">
                      <div className="text-right">
                        <div className="font-semibold">{f.origin.code}</div>
                        <div className="text-xs text-gray-500">{new Date(f.departure).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}</div>
                      </div>
                      <div className="text-center">
                        <div className="text-xs text-gray-400">{Math.floor(f.duration_minutes / 60)}h {f.duration_minutes % 60}m</div>
                        <div className="h-px w-10 bg-gray-300 dark:bg-gray-700 mx-auto" />
                      </div>
                      <div className="text-left">
                        <div className="font-semibold">{f.destination.code}</div>
                        <div className="text-xs text-gray-500">{new Date(f.arrival).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}</div>
                      </div>
                    </div>
                    <div className="flex items-center gap-4">
                      <div className="text-right">
                        <div className="text-lg font-bold">${cheapest ? cheapest.total.toFixed(2) : "—"}</div>
                        <div className="text-xs text-gray-500">per passenger</div>
                      </div>
                      <button className="btn-gradient h-10 px-6 rounded-md text-sm font-semibold text-white" onClick={() => select(f.flight_id)}>Select</button>
                    </div>
                  </CardContent>
                </Card>
              );
            })}
          </div>
        )}

        {!loading && results === null && !error && (
          <div className="rounded-lg border bg-white p-8 text-center shadow-sm dark:border-gray-700 dark:bg-gray-900">
            <PlaneTakeoff className="mx-auto mb-4 h-12 w-12 text-gray-300 dark:text-gray-600" />
            <h3 className="mb-2 text-lg font-semibold">Search for flights</h3>
            <p className="text-sm text-gray-500 dark:text-gray-400">Enter your origin, destination, and travel date to find available flights.</p>
          </div>
        )}
      </div>
    </div>
  );
}
