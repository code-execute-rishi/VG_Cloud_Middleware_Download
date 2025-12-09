const ProfileTab = ({ user }) => {
  const primaryEmail = user.emailAddresses?.[0]?.emailAddress || "Not set";
  const userId = user.id || user.externalId || "—";
  const createdAt = user.createdAt ? new Date(user.createdAt).toLocaleDateString() : "--";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="hidden md:block text-2xl md:text-3xl font-bold text-gray-900 mb-2">Profile</h1>
        <p className="text-gray-600 mb-4 md:mb-8 text-sm md:text-base">
          Manage your operator identity and fleet roles.
        </p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        <div className="rounded-xl bg-white border border-gray-200 shadow-sm p-6 flex items-center gap-4">
          <img
            src={user.imageUrl || "/placeholder-user.png"}
            alt={user.fullName || "User avatar"}
            className="h-20 w-20 rounded-full object-cover border border-gray-200"
          />
          <div className="space-y-1">
            <p className="text-lg font-semibold text-gray-900">{user.fullName || "Not set"}</p>
            <p className="text-sm text-gray-500">{primaryEmail}</p>
            <span className="inline-flex text-xs font-medium px-2 py-0.5 rounded-full bg-emerald-50 text-emerald-700">
              Flight Operations Lead
            </span>
          </div>
        </div>

        <div className="rounded-xl bg-white border border-gray-200 shadow-sm p-6 flex flex-col justify-between">
          <div>
            <p className="text-xs uppercase text-gray-500 tracking-wide">Operator ID</p>
            <p className="text-lg font-mono text-gray-900">{userId}</p>
          </div>
          <div>
            <p className="text-xs uppercase text-gray-500 tracking-wide">Member Since</p>
            <p className="text-lg text-gray-900">{createdAt}</p>
          </div>
        </div>

        <div className="rounded-xl bg-white border border-gray-200 shadow-sm p-6 flex flex-col gap-3">
          <p className="text-sm font-semibold text-gray-900">Access & Roles</p>
          <div className="space-y-2 text-sm text-gray-600">
            <div className="flex items-center justify-between">
              <span>Fleet Console</span>
              <span className="text-emerald-600 font-medium">Full Access</span>
            </div>
            <div className="flex items-center justify-between">
              <span>Mission Planning</span>
              <span className="text-emerald-600 font-medium">Full Access</span>
            </div>
            <div className="flex items-center justify-between">
              <span>Maintenance Logs</span>
              <span className="text-amber-600 font-medium">Review</span>
            </div>
            <div className="flex items-center justify-between">
              <span>Regulatory Docs</span>
              <span className="text-emerald-600 font-medium">Full Access</span>
            </div>
          </div>
          <button className="mt-auto inline-flex items-center justify-center rounded-lg bg-gray-900 text-white px-3 py-2 text-sm font-semibold hover:bg-gray-800 transition">
            Manage Access
          </button>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <div className="rounded-xl bg-white border border-gray-200 shadow-sm p-6 space-y-4">
          <h2 className="text-lg font-semibold text-gray-900">Contact Details</h2>
          <div className="space-y-3">
            <div>
              <p className="text-sm text-gray-500">Primary Email</p>
              <p className="text-gray-900 font-medium">{primaryEmail}</p>
            </div>
            <div>
              <p className="text-sm text-gray-500">Backup Email</p>
              <p className="text-gray-900 font-medium">{user.emailAddresses?.[1]?.emailAddress || "—"}</p>
            </div>
            <div>
              <p className="text-sm text-gray-500">Phone Number</p>
              <p className="text-gray-900 font-medium">{user.phoneNumbers?.[0]?.phoneNumber || "—"}</p>
            </div>
          </div>
        </div>

        <div className="rounded-xl bg-white border border-gray-200 shadow-sm p-6 space-y-4">
          <h2 className="text-lg font-semibold text-gray-900">Flight Credentials</h2>
          <div className="space-y-3 text-sm text-gray-600">
            <div className="flex items-center justify-between">
              <span>Licensing</span>
              <span className="font-medium text-gray-900">Part 107 Certified</span>
            </div>
            <div className="flex items-center justify-between">
              <span>BVLOS Clearance</span>
              <span className="font-medium text-emerald-600">Approved</span>
            </div>
            <div className="flex items-center justify-between">
              <span>Night Ops</span>
              <span className="font-medium text-emerald-600">Approved</span>
            </div>
            <div className="flex items-center justify-between">
              <span>Medical Certificate</span>
              <span className="font-medium text-gray-900">Valid</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default ProfileTab;

