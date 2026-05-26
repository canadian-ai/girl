import React from 'react';
import { useForm, Controller } from 'react-hook-form';
import { Button, Input, Select, Textarea, Spinner, Alert, Dialog } from './ui';
import { analytics } from './analytics';
import { useNavigate } from 'react-router-dom';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useAuth } from './auth';
import { useToast } from './toast';

interface ProjectFormData {
  name: string;
  description: string;
  team: string;
  category: string;
  priority: string;
  budget: string;
  startDate: string;
  endDate: string;
  notes: string;
}

interface Task {
  id: string;
  title: string;
  status: string;
  assignee: string;
}

const TEAMS = ['Engineering', 'Design', 'Marketing', 'Sales', 'Operations'];
const CATEGORIES = ['Frontend', 'Backend', 'Infrastructure', 'Mobile', 'AI/ML'];
const PRIORITIES = ['Low', 'Medium', 'High', 'Critical'];
const INITIAL_TASKS: Task[] = [
  { id: '1', title: 'Setup CI', status: 'done', assignee: 'alice' },
  { id: '2', title: 'Write tests', status: 'in-progress', assignee: 'bob' },
  { id: '3', title: 'Deploy staging', status: 'todo', assignee: 'charlie' },
  { id: '4', title: 'Security review', status: 'todo', assignee: 'dave' },
  { id: '5', title: 'Performance audit', status: 'todo', assignee: 'eve' },
];

export default function CreateProjectForm() {
  const navigate = useNavigate();
  const { user } = useAuth();
  const { toast } = useToast();
  const [tasks, setTasks] = React.useState<Task[]>(INITIAL_TASKS);
  const [showAdvanced, setShowAdvanced] = React.useState(false);
  const [isSubmitting, setIsSubmitting] = React.useState(false);
  const [activeTab, setActiveTab] = React.useState<'details' | 'tasks' | 'review'>('details');
  const [searchTerm, setSearchTerm] = React.useState('');

  const { control, handleSubmit, watch, formState: { errors } } = useForm<ProjectFormData>({
    defaultValues: {
      name: '',
      description: '',
      team: '',
      category: '',
      priority: 'Medium',
      budget: '',
      startDate: '',
      endDate: '',
      notes: '',
    }
  });

  const { data: projects, isLoading } = useQuery({
    queryKey: ['projects'],
    queryFn: () => fetch('/api/projects').then(r => r.json()),
  });

  const createProject = useMutation({
    mutationFn: (data: ProjectFormData) => fetch('/api/projects', { method: 'POST', body: JSON.stringify(data) }),
  });

  const watchTeam = watch('team');
  const watchCategory = watch('category');

  React.useEffect(() => {
    if (watchTeam) {
      analytics.track('team_selected', { team: watchTeam });
    }
  }, [watchTeam]);

  React.useEffect(() => {
    if (watchCategory) {
      analytics.track('category_selected', { category: watchCategory });
    }
  }, [watchCategory]);

  React.useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        navigate('/projects');
      }
      if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
        handleSubmit(onSubmit)();
      }
    };
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [navigate, handleSubmit]);

  const onSubmit = async (data: ProjectFormData) => {
    setIsSubmitting(true);
    try {
      analytics.track('project_created', { ...data, taskCount: tasks.length });
      await createProject.mutateAsync(data);
      toast({ title: 'Project created', description: 'Your project has been created successfully.' });
      navigate(`/projects/${data.name}`);
    } catch (err) {
      toast({ title: 'Error', description: 'Failed to create project.', variant: 'error' });
    } finally {
      setIsSubmitting(false);
    }
  };

  if (isLoading) {
    return <div className="flex items-center justify-center h-64"><Spinner /></div>;
  }

  return (
    <div className="max-w-4xl mx-auto p-6">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold">Create New Project</h1>
        <button onClick={() => navigate('/projects')} className="text-sm text-gray-500 hover:text-gray-700">
          Cancel
        </button>
      </div>

      <div className="mb-6">
        <div className="flex space-x-1 border-b">
          {(['details', 'tasks', 'review'] as const).map((tab) => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={`px-4 py-2 text-sm font-medium rounded-t ${
                activeTab === tab ? 'bg-white border-l border-r border-t -mb-px' : 'text-gray-500 hover:text-gray-700'
              }`}
            >
              {tab === 'details' ? 'Project Details' : tab === 'tasks' ? 'Tasks' : 'Review'}
            </button>
          ))}
        </div>
      </div>

      <form onSubmit={handleSubmit(onSubmit)}>
        {activeTab === 'details' && (
          <div className="space-y-4">
            <Controller
              name="name"
              control={control}
              rules={{ required: 'Name is required' }}
              render={({ field }) => (
                <div>
                  <label className="block text-sm font-medium mb-1">Project Name</label>
                  <Input {...field} placeholder="Enter project name" error={errors.name?.message} />
                </div>
              )}
            />

            <Controller
              name="description"
              control={control}
              render={({ field }) => (
                <div>
                  <label className="block text-sm font-medium mb-1">Description</label>
                  <Textarea {...field} placeholder="Describe your project" rows={4} />
                </div>
              )}
            />

            <Controller
              name="team"
              control={control}
              render={({ field }) => (
                <div>
                  <label className="block text-sm font-medium mb-1">Team</label>
                  <Select {...field} placeholder="Select a team">
                    {TEAMS.map((team) => (
                      <option key={team} value={team}>{team}</option>
                    ))}
                  </Select>
                </div>
              )}
            />

            <Controller
              name="category"
              control={control}
              render={({ field }) => (
                <div>
                  <label className="block text-sm font-medium mb-1">Category</label>
                  <Select {...field} placeholder="Select a category">
                    {CATEGORIES.map((cat) => (
                      <option key={cat} value={cat}>{cat}</option>
                    ))}
                  </Select>
                </div>
              )}
            />

            <Controller
              name="priority"
              control={control}
              render={({ field }) => (
                <div>
                  <label className="block text-sm font-medium mb-1">Priority</label>
                  <Select {...field}>
                    {PRIORITIES.map((p) => (
                      <option key={p} value={p}>{p}</option>
                    ))}
                  </Select>
                </div>
              )}
            />

            <button
              type="button"
              onClick={() => setShowAdvanced(!showAdvanced)}
              className="text-sm text-blue-600 hover:text-blue-800"
            >
              {showAdvanced ? 'Hide' : 'Show'} Advanced Options
            </button>

            {showAdvanced && (
              <div className="space-y-4 p-4 bg-gray-50 rounded">
                <Controller
                  name="budget"
                  control={control}
                  render={({ field }) => (
                    <div>
                      <label className="block text-sm font-medium mb-1">Budget</label>
                      <Input {...field} type="number" placeholder="Enter budget" />
                    </div>
                  )}
                />

                <Controller
                  name="startDate"
                  control={control}
                  render={({ field }) => (
                    <div>
                      <label className="block text-sm font-medium mb-1">Start Date</label>
                      <Input {...field} type="date" />
                    </div>
                  )}
                />

                <Controller
                  name="endDate"
                  control={control}
                  render={({ field }) => (
                    <div>
                      <label className="block text-sm font-medium mb-1">End Date</label>
                      <Input {...field} type="date" />
                    </div>
                  )}
                />

                <Controller
                  name="notes"
                  control={control}
                  render={({ field }) => (
                    <div>
                      <label className="block text-sm font-medium mb-1">Notes</label>
                      <Textarea {...field} placeholder="Additional notes" rows={3} />
                    </div>
                  )}
                />
              </div>
            )}
          </div>
        )}

        {activeTab === 'tasks' && (
          <div className="space-y-4">
            <div className="flex items-center space-x-2 mb-4">
              <Input
                placeholder="Search tasks..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
              />
            </div>

            <div className="space-y-2">
              {tasks
                .filter((t) => t.title.toLowerCase().includes(searchTerm.toLowerCase()))
                .map((task) => (
                  <div key={task.id} className="flex items-center justify-between p-3 border rounded">
                    <div>
                      <p className="font-medium">{task.title}</p>
                      <p className="text-sm text-gray-500">{task.assignee}</p>
                    </div>
                    <div className="flex items-center space-x-2">
                      <span className={`px-2 py-1 text-xs rounded ${
                        task.status === 'done' ? 'bg-green-100 text-green-800' : 
                        task.status === 'in-progress' ? 'bg-blue-100 text-blue-800' : 
                        'bg-gray-100 text-gray-800'
                      }`}>
                        {task.status}
                      </span>
                      <button
                        onClick={() => {
                          setTasks(tasks.filter((t) => t.id !== task.id));
                        }}
                        className="text-red-500 hover:text-red-700 text-sm"
                      >
                        Remove
                      </button>
                    </div>
                  </div>
                ))}
            </div>
          </div>
        )}

        {activeTab === 'review' && (
          <div className="space-y-4">
            <Alert variant="info">
              Review your project details before submitting. You can go back to make changes.
            </Alert>

            <div className="bg-gray-50 p-4 rounded space-y-2">
              {Object.entries(watch()).filter(([_, v]) => v).map(([key, value]) => (
                <div key={key} className="flex justify-between">
                  <span className="font-medium capitalize">{key}:</span>
                  <span>{String(value)}</span>
                </div>
              ))}
              <div className="flex justify-between">
                <span className="font-medium">Tasks:</span>
                <span>{tasks.length} total</span>
              </div>
              <div className="flex justify-between">
                <span className="font-medium">Team members:</span>
                <span>{user?.email || 'Unknown'}</span>
              </div>
            </div>

            {errors.name && (
              <Alert variant="error">{errors.name.message}</Alert>
            )}
          </div>
        )}

        <div className="flex justify-end space-x-3 mt-8 pt-4 border-t">
          <Button type="button" variant="secondary" onClick={() => navigate('/projects')}>
            Cancel
          </Button>
          <Button type="submit" disabled={isSubmitting}>
            {isSubmitting ? <Spinner /> : 'Create Project'}
          </Button>
        </div>
      </form>
    </div>
  );
}
