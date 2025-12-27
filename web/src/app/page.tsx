export default function Home() {
  return (
    <main className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100">
      <div className="container mx-auto px-4 py-16">
        {/* Header */}
        <header className="text-center mb-16">
          <h1 className="text-6xl font-bold text-gray-900 mb-6">
            BitGo Wallets
          </h1>
          <p className="text-xl text-gray-600 max-w-3xl mx-auto">
            Secure multi-signature wallet management platform for
            enterprise-grade cryptocurrency storage and transactions
          </p>
        </header>

        {/* Hero Section */}
        <section className="mb-20">
          <div className="bg-white rounded-2xl shadow-xl p-8 md:p-12">
            <div className="grid md:grid-cols-2 gap-12 items-center">
              <div>
                <h2 className="text-3xl font-bold text-gray-900 mb-6">
                  Enterprise Security, Simplified
                </h2>
                <p className="text-gray-600 mb-8 leading-relaxed">
                  Manage your cryptocurrency assets with institutional-grade
                  security. Our multi-signature wallet solution provides
                  unparalleled protection while maintaining ease of use for your
                  team.
                </p>
                <div className="flex flex-col sm:flex-row gap-4">
                  <button className="bg-blue-600 hover:bg-blue-700 text-white px-8 py-4 rounded-lg font-semibold transition-colors">
                    Get Started
                  </button>
                  <button className="border-2 border-gray-300 hover:border-gray-400 text-gray-700 px-8 py-4 rounded-lg font-semibold transition-colors">
                    Learn More
                  </button>
                </div>
              </div>
              <div className="bg-gradient-to-br from-blue-600 to-indigo-700 rounded-xl h-64 flex items-center justify-center">
                <div className="text-white text-center">
                  <div className="text-4xl mb-4">🔐</div>
                  <div className="text-lg font-semibold">Secure Wallet</div>
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* Features Section */}
        <section className="mb-20">
          <h2 className="text-4xl font-bold text-center text-gray-900 mb-12">
            Key Features
          </h2>
          <div className="grid md:grid-cols-3 gap-8">
            <div className="bg-white p-8 rounded-xl shadow-lg">
              <div className="text-3xl mb-4">🛡️</div>
              <h3 className="text-xl font-bold text-gray-900 mb-4">
                Multi-Signature Security
              </h3>
              <p className="text-gray-600">
                Require multiple approvals for transactions, ensuring no single
                point of failure in your security model.
              </p>
            </div>
            <div className="bg-white p-8 rounded-xl shadow-lg">
              <div className="text-3xl mb-4">⚡</div>
              <h3 className="text-xl font-bold text-gray-900 mb-4">
                Fast Transactions
              </h3>
              <p className="text-gray-600">
                Process cryptocurrency transactions quickly while maintaining
                the highest security standards.
              </p>
            </div>
            <div className="bg-white p-8 rounded-xl shadow-lg">
              <div className="text-3xl mb-4">📊</div>
              <h3 className="text-xl font-bold text-gray-900 mb-4">
                Advanced Analytics
              </h3>
              <p className="text-gray-600">
                Monitor your portfolio with comprehensive analytics and
                reporting tools for better decision making.
              </p>
            </div>
          </div>
        </section>

        {/* CTA Section */}
        <section className="text-center">
          <div className="bg-gradient-to-r from-blue-600 to-indigo-700 text-white p-12 rounded-2xl">
            <h2 className="text-3xl font-bold mb-6">
              Ready to Secure Your Assets?
            </h2>
            <p className="text-xl mb-8 opacity-90">
              Join thousands of businesses trusting BitGo Wallets with their
              cryptocurrency management
            </p>
            <button className="bg-white text-blue-600 px-8 py-4 rounded-lg font-semibold hover:bg-gray-100 transition-colors">
              Start Your Free Trial
            </button>
          </div>
        </section>
      </div>
    </main>
  );
}
