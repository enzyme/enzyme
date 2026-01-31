import { useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { useCreateInvite } from '../hooks/useWorkspaces';
import { Button, Input, toast } from '../components/ui';

export function InvitePage() {
  const { workspaceId } = useParams<{ workspaceId: string }>();
  const createInvite = useCreateInvite(workspaceId!);
  const [inviteLink, setInviteLink] = useState<string | null>(null);
  const [expiresIn, setExpiresIn] = useState('7');
  const [maxUses, setMaxUses] = useState('');

  const handleCreateInvite = async () => {
    try {
      const result = await createInvite.mutateAsync({
        expires_in_days: parseInt(expiresIn) || undefined,
        max_uses: maxUses ? parseInt(maxUses) : undefined,
      });
      const link = `${window.location.origin}/invites/${result.invite.code}`;
      setInviteLink(link);
      toast('Invite link created!', 'success');
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Failed to create invite', 'error');
    }
  };

  const handleCopyLink = () => {
    if (inviteLink) {
      navigator.clipboard.writeText(inviteLink);
      toast('Link copied to clipboard', 'success');
    }
  };

  return (
    <div className="flex-1 overflow-auto bg-white dark:bg-gray-900">
      <div className="max-w-2xl mx-auto p-8">
        <div className="flex items-center justify-between mb-8">
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
            Invite People
          </h1>
          <Link
            to={`/workspaces/${workspaceId}`}
            className="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
          >
            <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </Link>
        </div>

        <div className="bg-gray-50 dark:bg-gray-800 rounded-lg p-6 space-y-6">
          <p className="text-gray-600 dark:text-gray-300">
            Create an invite link to share with people you want to join this workspace.
          </p>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Expires in (days)
              </label>
              <select
                value={expiresIn}
                onChange={(e) => setExpiresIn(e.target.value)}
                className="w-full bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-lg px-3 py-2 text-gray-900 dark:text-white"
              >
                <option value="1">1 day</option>
                <option value="7">7 days</option>
                <option value="30">30 days</option>
                <option value="">Never</option>
              </select>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Max uses
              </label>
              <Input
                type="number"
                value={maxUses}
                onChange={(e) => setMaxUses(e.target.value)}
                placeholder="Unlimited"
                min="1"
              />
            </div>
          </div>

          <Button onClick={handleCreateInvite} isLoading={createInvite.isPending}>
            Generate Invite Link
          </Button>

          {inviteLink && (
            <div className="mt-6 p-4 bg-white dark:bg-gray-700 rounded-lg border border-gray-200 dark:border-gray-600">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                Your invite link
              </label>
              <div className="flex gap-2">
                <input
                  type="text"
                  value={inviteLink}
                  readOnly
                  className="flex-1 bg-gray-100 dark:bg-gray-600 border border-gray-300 dark:border-gray-500 rounded-lg px-3 py-2 text-gray-900 dark:text-white text-sm"
                />
                <Button onClick={handleCopyLink} variant="secondary">
                  Copy
                </Button>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
