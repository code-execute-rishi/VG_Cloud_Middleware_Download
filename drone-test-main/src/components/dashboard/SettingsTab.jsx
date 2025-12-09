const SettingsTab = () => {
  return (
    <div>
      <h1 className="hidden md:block text-2xl md:text-3xl font-bold text-gray-900 mb-2">Settings</h1>
      <p className="text-gray-600 mb-4 md:mb-8 text-sm md:text-base">
        Configure your preferences
      </p>
      <div className="space-y-3 md:space-y-4">
        <div className="rounded-lg bg-white border border-gray-200 p-4 md:p-6 shadow-sm">
          <h2 className="text-lg md:text-xl font-semibold text-gray-900 mb-3 md:mb-4">Account Settings</h2>
          <div className="space-y-3">
            <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-3">
              <div className="flex-1">
                <p className="text-gray-900 font-medium">Notifications</p>
                <p className="text-sm text-gray-500">Manage notification preferences</p>
              </div>
              <button className="px-4 py-2 bg-gray-900 hover:bg-gray-800 rounded-lg text-white text-sm w-full md:w-auto">
                Configure
              </button>
            </div>
            <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-3">
              <div className="flex-1">
                <p className="text-gray-900 font-medium">Privacy</p>
                <p className="text-sm text-gray-500">Control your privacy settings</p>
              </div>
              <button className="px-4 py-2 bg-gray-900 hover:bg-gray-800 rounded-lg text-white text-sm w-full md:w-auto">
                Configure
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default SettingsTab;



