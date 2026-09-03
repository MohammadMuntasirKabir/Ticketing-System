import { Search, MapPin, Clock } from "lucide-react";
import Link from "next/link";

export default function LandingPage() {
  return (
    <div className="flex min-h-screen flex-col">
      <main className="flex-1">
        <section className="relative overflow-hidden bg-brand-gradient text-white">
          <div className="container relative mx-auto px-4 py-24 md:py-36">
            <div className="mx-auto max-w-3xl text-center">
              <span className="mb-6 inline-flex items-center gap-2 rounded-full border border-white/25 bg-white/10 px-4 py-1.5 text-sm font-medium backdrop-blur">
                Book with confidence
              </span>
              <h1 className="text-4xl font-extrabold leading-tight tracking-tight sm:text-5xl md:text-6xl">
                Find your perfect flight,{" "}
                <span className="underline decoration-white/40 decoration-4 underline-offset-8">
                  every time
                </span>
              </h1>
              <p className="mx-auto mt-6 max-w-2xl text-lg text-white/90 md:text-xl">
                Search, book, manage, and check in — all in one place.
              </p>
              <div className="mt-9 flex flex-col gap-4 sm:flex-row sm:justify-center">
                <Link href="/search" className="btn-gradient inline-flex items-center justify-center rounded-xl px-7 py-3.5 text-base font-semibold text-white shadow-xl transition hover:brightness-110 active:scale-95">
                  Search Flights
                </Link>
                <Link href="/login" className="inline-flex items-center justify-center rounded-xl border border-white/40 bg-white/10 px-7 py-3.5 text-base font-semibold text-white backdrop-blur transition hover:bg-white/20 active:scale-95">
                  Sign In
                </Link>
              </div>
            </div>
          </div>
        </section>

        <section className="py-16 md:py-24">
          <div className="container mx-auto px-4">
            <div className="mx-auto mb-14 max-w-2xl text-center">
              <h2 className="text-3xl font-extrabold tracking-tight text-gray-900 dark:text-gray-100">
                Everything you need for{" "}
                <span className="text-brand-gradient">air travel</span>
              </h2>
              <p className="mt-3 text-gray-600 dark:text-gray-400">
                From booking to boarding — seamless, fast, and built for the modern traveler.
              </p>
            </div>
            <div className="grid gap-6 md:grid-cols-3">
              {[
                { icon: Search, title: "Flight Search", desc: "Find the best flights across airlines with real-time pricing and availability." },
                { icon: Clock, title: "Easy Booking", desc: "Select seats, add passengers, and complete your booking in minutes." },
                { icon: MapPin, title: "Online Check-in", desc: "Check in from your phone, choose your seat, and get your boarding pass." },
              ].map((f) => (
                <div key={f.title} className="rounded-2xl border border-gray-200 bg-white p-7 transition-all duration-300 hover:-translate-y-1 hover:shadow-xl dark:border-gray-700 dark:bg-gray-900">
                  <div className="mb-5 flex h-12 w-12 items-center justify-center rounded-xl bg-brand-gradient text-white shadow-md shadow-indigo-500/25">
                    <f.icon className="h-6 w-6" />
                  </div>
                  <h3 className="mb-2 text-xl font-semibold text-gray-900 dark:text-gray-100">{f.title}</h3>
                  <p className="text-sm leading-relaxed text-gray-600 dark:text-gray-400">{f.desc}</p>
                </div>
              ))}
            </div>
          </div>
        </section>
      </main>
    </div>
  );
}