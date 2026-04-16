'use client'

import { useState, useRef, useEffect } from 'react'
import Link from 'next/link'
import { useParams, useRouter } from 'next/navigation'
import { useRepair, useVehicleSystemLabel, useNotes } from '@/lib/queries'
import { useApi, type RepairStep, type Part, type NoteWithAuthor } from '@/lib/api'
import { useQueryClient } from '@tanstack/react-query'

const REPAIR_STATUS_LABEL: Record<string, string> = {
  draft:         'Draft',
  in_progress:   'In progress',
  pending_parts: 'Pending parts',
  completed:     'Completed',
}

const REPAIR_STATUS_STYLE: Record<string, string> = {
  draft:         'bg-zinc-800 text-zinc-400',
  in_progress:   'bg-blue-950 text-blue-400',
  pending_parts: 'bg-amber-950 text-amber-400',
  completed:     'bg-green-950 text-green-400',
}

const PART_STATUS_NEXT: Record<string, string> = {
  needed:    'ordered',
  ordered:   'in_stock',
  in_stock:  'installed',
  installed: 'installed',
}

const PART_STATUS_LABEL: Record<string, string> = {
  needed:    'Needed',
  ordered:   'Ordered',
  in_stock:  'In stock',
  installed: 'Installed',
}

const PART_STATUS_STYLE: Record<string, string> = {
  needed:    'bg-zinc-800 text-zinc-400',
  ordered:   'bg-blue-950 text-blue-400',
  in_stock:  'bg-amber-950 text-amber-400',
  installed: 'bg-green-950 text-green-400',
}

export default function RepairCardPage() {
  const params = useParams()
  const vehicleId = params.vehicleId as string
  const system    = params.system as string
  const repairId  = params.repairId as string

  const { data: repair, isLoading, error } = useRepair(repairId)
  const { data: notes, isLoading: notesLoading } = useNotes(repairId)
  const systemLabel = useVehicleSystemLabel(vehicleId, system)

  const api = useApi()
  const qc  = useQueryClient()

  const [togglingStep,    setTogglingStep]    = useState<string | null>(null)
  const [advancingPart,   setAdvancingPart]   = useState<string | null>(null)
  const [transitioning,   setTransitioning]   = useState(false)
  const [noteModalOpen,   setNoteModalOpen]   = useState(false)
  const [noteBody,        setNoteBody]        = useState('')
  const [submittingNote,  setSubmittingNote]  = useState(false)
  const [deleteModalOpen, setDeleteModalOpen] = useState(false)
  const [deleting,        setDeleting]        = useState(false)
  const [stepModalOpen,   setStepModalOpen]   = useState(false)
  const [partModalOpen,   setPartModalOpen]   = useState(false)
  const router = useRouter()

  if (isLoading) return <LoadingState />
  if (error)     return <ErrorState message={(error as Error).message} />
  if (!repair)   return <ErrorState message="Repair not found." />

  const steps      = (repair.steps  ?? []) as RepairStep[]
  const parts      = (repair.parts  ?? []) as Part[]
  const status     = repair.status ?? 'draft'
  const doneSteps  = steps.filter(s => s.completed).length
  const totalSteps = steps.length
  const progressPct = totalSteps > 0 ? Math.round((doneSteps / totalSteps) * 100) : 0

  const canStart    = status === 'draft'
  const canComplete = status === 'in_progress' || status === 'pending_parts'
  const isCompleted = status === 'completed'
  const isEditable  = status === 'draft' || status === 'in_progress'

  async function toggleStep(stepId: string, currentlyDone: boolean) {
    if (togglingStep) return
    setTogglingStep(stepId)
    try {
      await api.repairs.toggleStep(repairId, stepId)
      await qc.invalidateQueries({ queryKey: ['repair', repairId] })
    } catch (e) { console.error(e) }
    finally { setTogglingStep(null) }
  }

  async function advancePart(partId: string, currentStatus: string) {
    if (advancingPart || currentStatus === 'installed') return
    setAdvancingPart(partId)
    try {
      const next = PART_STATUS_NEXT[currentStatus] ?? 'ordered'
      await api.repairs.updatePartStatus(repairId, partId, next)
      await qc.invalidateQueries({ queryKey: ['repair', repairId] })
    } catch (e) { console.error(e) }
    finally { setAdvancingPart(null) }
  }

  async function transitionRepair(toStatus: string) {
    if (transitioning) return
    setTransitioning(true)
    try {
      await api.repairs.updateStatus(repairId, toStatus)
      await qc.invalidateQueries({ queryKey: ['repair', repairId] })
    } catch (e) { console.error(e) }
    finally { setTransitioning(false) }
  }

  async function submitNote() {
    if (!noteBody.trim() || submittingNote) return
    setSubmittingNote(true)
    try {
      await api.repairs.createNote(repairId, noteBody.trim())
      await qc.invalidateQueries({ queryKey: ['repair-notes', repairId] })
      setNoteBody('')
      setNoteModalOpen(false)
    } catch (e) { console.error(e) }
    finally { setSubmittingNote(false) }
  }

  async function deleteRepair() {
    if (deleting) return
    setDeleting(true)
    try {
      await api.repairs.deleteRepair(repairId)
      router.push(`/vehicles/${vehicleId}/${system}`)
    } catch (e) { console.error(e) }
    finally { setDeleting(false) }
  }

  return (
    <div className="min-h-screen bg-black text-white">
      <header className="bg-zinc-950 border-b border-zinc-800 sticky top-0 z-10">
        <div className="max-w-2xl mx-auto px-6 py-4">
          <nav className="flex items-center gap-1.5 text-xs text-zinc-500 mb-3 flex-wrap">
            <Link href="/" className="hover:text-zinc-300 transition-colors">Garage</Link>
            <Chevron />
            <Link href={`/vehicles/${vehicleId}`} className="hover:text-zinc-300 transition-colors">Vehicle</Link>
            <Chevron />
            <Link href={`/vehicles/${vehicleId}/${system}`} className="hover:text-zinc-300 transition-colors">{systemLabel}</Link>
            <Chevron />
            <span className="text-zinc-200 truncate max-w-40">{repair.title}</span>
          </nav>

          <div className="flex items-start justify-between gap-4">
            <div>
              <h1 className="text-xl font-semibold text-white leading-tight">{repair.title}</h1>
              {repair.started_at && (
                <p className="text-xs text-zinc-500 mt-1">
                  Started {new Date(repair.started_at).toLocaleDateString()}
                  {repair.completed_at && <> · Completed {new Date(repair.completed_at).toLocaleDateString()}</>}
                </p>
              )}
            </div>
            <span className={`text-xs px-2.5 py-1 rounded-full font-medium flex-shrink-0 mt-0.5 ${REPAIR_STATUS_STYLE[status] ?? REPAIR_STATUS_STYLE.draft}`}>
              {REPAIR_STATUS_LABEL[status] ?? status}
            </span>
          </div>

          {status === 'in_progress' && totalSteps > 0 && (
            <div className="mt-3">
              <div className="flex justify-between text-xs text-zinc-500 mb-1.5">
                <span>{doneSteps} of {totalSteps} steps done</span>
                <span>{progressPct}%</span>
              </div>
              <div className="h-1 bg-zinc-800 rounded-full overflow-hidden">
                <div
                  className="h-full bg-blue-500 rounded-full transition-all duration-500"
                  style={{ width: `${progressPct}%` }}
                />
              </div>
            </div>
          )}
        </div>
      </header>

      <div className="max-w-2xl mx-auto px-6 py-6 space-y-3">

        {canStart && (
          <button
            onClick={() => transitionRepair('in_progress')}
            disabled={transitioning}
            className="w-full bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white font-medium text-sm py-3 rounded-xl transition-colors"
          >
            {transitioning ? 'Starting…' : 'Start repair'}
          </button>
        )}

        {/* Steps section */}
        <Section
          title="Steps"
          count={totalSteps > 0 ? `${doneSteps}/${totalSteps}` : undefined}
          accent={doneSteps === totalSteps && totalSteps > 0}
          action={
            isEditable ? (
              <button
                onClick={() => setStepModalOpen(true)}
                className="text-xs text-blue-400 hover:text-blue-300 font-medium transition-colors"
              >
                + Add step
              </button>
            ) : undefined
          }
        >
          {steps.length === 0 ? (
            <div className="px-4 py-6 text-sm text-zinc-600 text-center">
              No steps yet.{isEditable && ' Tap "+ Add step" to build out the procedure.'}
            </div>
          ) : (
            <div className="divide-y divide-zinc-800">
              {steps.map((step, i) => (
                <button
                  key={step.id}
                  onClick={() => toggleStep(step.id!, step.completed ?? false)}
                  disabled={!!togglingStep || isCompleted}
                  className={`w-full text-left px-4 py-3.5 flex items-start gap-3.5 transition-colors
                    ${step.completed ? 'bg-green-950/30' : 'hover:bg-zinc-800/50'}
                    ${togglingStep === step.id ? 'opacity-40' : ''}`}
                >
                  <div className={`flex-shrink-0 w-6 h-6 rounded-full border flex items-center justify-center text-xs font-semibold mt-0.5 transition-all
                    ${step.completed
                      ? 'bg-green-500 border-green-500 text-white'
                      : 'border-zinc-600 text-zinc-500'}`}
                  >
                    {step.completed ? <CheckIcon /> : <span>{i + 1}</span>}
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className={`text-sm font-medium leading-snug transition-colors
                      ${step.completed ? 'text-zinc-600 line-through' : 'text-zinc-100'}`}>
                      {step.title}
                    </p>
                    {step.notes && !step.completed && (
                      <p className="text-xs text-zinc-500 mt-0.5 leading-relaxed">{step.notes}</p>
                    )}
                    {step.torque_spec && !step.completed && (
                      <span className="inline-block mt-1.5 text-xs bg-amber-950 text-amber-400 px-2 py-0.5 rounded font-mono font-medium">
                        {step.torque_spec}
                      </span>
                    )}
                    {step.completed && step.completed_at && (
                      <p className="text-xs text-zinc-600 mt-0.5">
                        Done {new Date(step.completed_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                      </p>
                    )}
                  </div>
                </button>
              ))}
            </div>
          )}
        </Section>

        {/* Parts section */}
        <Section
          title="Parts"
          count={parts.length > 0 ? `${parts.filter(p => p.status === 'installed').length}/${parts.length} installed` : undefined}
          action={
            isEditable ? (
              <button
                onClick={() => setPartModalOpen(true)}
                className="text-xs text-blue-400 hover:text-blue-300 font-medium transition-colors"
              >
                + Add part
              </button>
            ) : undefined
          }
        >
          {parts.length === 0 ? (
            <div className="px-4 py-6 text-sm text-zinc-600 text-center">
              No parts yet.{isEditable && ' Tap "+ Add part" to track what you need.'}
            </div>
          ) : (
            <>
              <div className="divide-y divide-zinc-800">
                {parts.map(part => {
                  const pStatus     = part.status ?? 'needed'
                  const isInstalled = pStatus === 'installed'
                  const isAdvancing = advancingPart === part.id
                  return (
                    <button
                      key={part.id}
                      onClick={() => advancePart(part.id!, pStatus)}
                      disabled={isInstalled || !!advancingPart || isCompleted}
                      className={`w-full text-left px-4 py-3.5 flex items-center gap-3 transition-colors
                        ${isInstalled ? 'opacity-40' : 'hover:bg-zinc-800/50'}
                        ${isAdvancing ? 'opacity-40' : ''}`}
                    >
                      <div className="flex-1 min-w-0">
                        <p className={`text-sm font-medium ${isInstalled ? 'text-zinc-600 line-through' : 'text-zinc-100'}`}>
                          {part.name}
                        </p>
                        <div className="flex items-center gap-2 mt-0.5">
                          {part.part_number && (
                            <span className="text-xs text-zinc-500 font-mono">{part.part_number}</span>
                          )}
                          {part.supplier && (
                            <span className="text-xs text-zinc-600">{part.supplier}</span>
                          )}
                        </div>
                      </div>
                      <div className="flex items-center gap-2 flex-shrink-0">
                        {(part.cost ?? 0) > 0 && (
                          <span className="text-xs text-zinc-400 font-medium">${part.cost}</span>
                        )}
                        <span className={`text-xs px-2 py-0.5 rounded-full font-medium ${PART_STATUS_STYLE[pStatus] ?? PART_STATUS_STYLE.needed}`}>
                          {PART_STATUS_LABEL[pStatus] ?? pStatus}
                        </span>
                        {!isInstalled && (
                          <svg className="w-3.5 h-3.5 text-zinc-600" fill="none" stroke="currentColor" strokeWidth={2} viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" d="M8.25 4.5l7.5 7.5-7.5 7.5" />
                          </svg>
                        )}
                      </div>
                    </button>
                  )
                })}
              </div>
              {(repair.total_parts_cost ?? 0) > 0 && (
                <div className="px-4 py-3 border-t border-zinc-800 flex justify-between text-sm">
                  <span className="text-zinc-500">Total parts cost</span>
                  <span className="font-semibold text-zinc-100">${repair.total_parts_cost}</span>
                </div>
              )}
            </>
          )}
        </Section>

        {/* Notes section */}
        <Section
          title="Notes"
          count={notes && notes.length > 0 ? `${notes.length}` : undefined}
          action={
            <button
              onClick={() => setNoteModalOpen(true)}
              className="text-xs text-blue-400 hover:text-blue-300 font-medium transition-colors"
            >
              + Add note
            </button>
          }
        >
          {notesLoading ? (
            <div className="px-4 py-6 text-sm text-zinc-600 text-center">Loading…</div>
          ) : !notes || notes.length === 0 ? (
            <div className="px-4 py-6 text-sm text-zinc-600 text-center">
              No notes yet.{!isCompleted && ' Tap "+ Add note" to log something.'}
            </div>
          ) : (
            <div className="divide-y divide-zinc-800">
              {(notes as NoteWithAuthor[]).map(note => (
                <div key={note.id} className="px-4 py-3.5">
                  <p className="text-sm text-zinc-200 leading-relaxed">{note.body}</p>
                  <div className="flex items-center gap-2 mt-1.5">
                    {note.author_name && (
                      <span className="text-xs text-zinc-500">{note.author_name}</span>
                    )}
                    {note.created_at && (
                      <span className="text-xs text-zinc-700">
                        {new Date(note.created_at).toLocaleString([], {
                          month: 'short', day: 'numeric',
                          hour: '2-digit', minute: '2-digit',
                        })}
                      </span>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </Section>

        {/* Status transitions */}
        {canComplete && !isCompleted && (
          <div className="flex gap-2 pt-1">
            {status === 'in_progress' && (
              <button
                onClick={() => transitionRepair('pending_parts')}
                disabled={transitioning}
                className="flex-1 border border-zinc-700 text-zinc-400 hover:bg-zinc-800 disabled:opacity-50 font-medium text-sm py-3 rounded-xl transition-colors"
              >
                Waiting on parts
              </button>
            )}
            {status === 'pending_parts' && (
              <button
                onClick={() => transitionRepair('in_progress')}
                disabled={transitioning}
                className="flex-1 border border-blue-800 text-blue-400 hover:bg-blue-950 disabled:opacity-50 font-medium text-sm py-3 rounded-xl transition-colors"
              >
                Parts arrived, resume
              </button>
            )}
            <button
              onClick={() => transitionRepair('completed')}
              disabled={transitioning}
              className="flex-1 bg-green-700 hover:bg-green-600 disabled:opacity-50 text-white font-medium text-sm py-3 rounded-xl transition-colors"
            >
              {transitioning ? 'Saving…' : 'Mark complete'}
            </button>
          </div>
        )}

        {isCompleted && (
          <div className="space-y-2">
            <div className="rounded-xl border border-green-900 bg-green-950/50 px-4 py-4 text-center">
              <div className="text-green-400 font-medium text-sm">Repair completed</div>
              {repair.completed_at && (
                <div className="text-green-600 text-xs mt-0.5">
                  {new Date(repair.completed_at).toLocaleDateString([], {
                    month: 'long', day: 'numeric', year: 'numeric',
                  })}
                </div>
              )}
            </div>
            <div className="flex gap-2">
              <button
                onClick={() => transitionRepair('in_progress')}
                disabled={transitioning}
                className="flex-1 border border-zinc-700 text-zinc-400 hover:bg-zinc-800 disabled:opacity-50 font-medium text-sm py-3 rounded-xl transition-colors"
              >
                {transitioning ? 'Reopening…' : 'Reopen repair'}
              </button>
              <button
                onClick={() => setDeleteModalOpen(true)}
                className="border border-red-900 text-red-500 hover:bg-red-950 font-medium text-sm py-3 px-4 rounded-xl transition-colors"
              >
                Delete
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Floating note button */}
      {!noteModalOpen && !stepModalOpen && !partModalOpen && (
        <button
          onClick={() => setNoteModalOpen(true)}
          className="fixed bottom-6 right-6 w-14 h-14 bg-zinc-800 hover:bg-zinc-700 text-white rounded-full shadow-xl border border-zinc-700 flex items-center justify-center transition-colors z-20"
          aria-label="Add note"
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" strokeWidth={1.8} viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" d="M16.862 4.487l1.687-1.688a1.875 1.875 0 112.652 2.652L10.582 16.07a4.5 4.5 0 01-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 011.13-1.897l8.932-8.931zm0 0L19.5 7.125M18 14v4.75A2.25 2.25 0 0115.75 21H5.25A2.25 2.25 0 013 18.75V8.25A2.25 2.25 0 015.25 6H10" />
          </svg>
        </button>
      )}

      {deleteModalOpen && (
        <ConfirmDeleteModal
          onClose={() => setDeleteModalOpen(false)}
          onConfirm={deleteRepair}
          confirming={deleting}
        />
      )}

      {noteModalOpen && (
        <NoteModal
          body={noteBody}
          onChange={setNoteBody}
          onSubmit={submitNote}
          onClose={() => { setNoteModalOpen(false); setNoteBody('') }}
          submitting={submittingNote}
        />
      )}

      {stepModalOpen && (
        <AddStepModal
          repairId={repairId}
          nextPosition={steps.length + 1}
          onClose={() => setStepModalOpen(false)}
          onSaved={async () => {
            await qc.invalidateQueries({ queryKey: ['repair', repairId] })
            setStepModalOpen(false)
          }}
        />
      )}

      {partModalOpen && (
        <AddPartModal
          repairId={repairId}
          onClose={() => setPartModalOpen(false)}
          onSaved={async () => {
            await qc.invalidateQueries({ queryKey: ['repair', repairId] })
            setPartModalOpen(false)
          }}
        />
      )}
    </div>
  )
}

// ── Add Step Modal ────────────────────────────────────────────────────────────

function AddStepModal({
  repairId,
  nextPosition,
  onClose,
  onSaved,
}: {
  repairId: string
  nextPosition: number
  onClose: () => void
  onSaved: () => Promise<void>
}) {
  const api = useApi()
  const [title,      setTitle]      = useState('')
  const [notes,      setNotes]      = useState('')
  const [torqueSpec, setTorqueSpec] = useState('')
  const [saving,     setSaving]     = useState(false)
  const [err,        setErr]        = useState('')

  async function submit() {
    if (!title.trim()) { setErr('A title is required.'); return }
    setSaving(true)
    setErr('')
    try {
      await api.repairs.createStep(repairId, {
        position:    nextPosition,
        title:       title.trim(),
        notes:       notes.trim() || undefined,
        torque_spec: torqueSpec.trim() || undefined,
      })
      await onSaved()
    } catch (e: any) {
      setErr(e.message ?? 'Something went wrong.')
      setSaving(false)
    }
  }

  return (
    <DarkModal title="Add step" onClose={onClose}>
      <div className="space-y-3">
        <div>
          <label className="block text-xs text-zinc-500 mb-1">
            Title <span className="text-red-400">*</span>
          </label>
          <input
            type="text"
            value={title}
            onChange={e => setTitle(e.target.value)}
            placeholder="e.g. Remove drain plug"
            autoFocus
            className="w-full bg-zinc-800 border border-zinc-700 rounded-xl px-3.5 py-2.5 text-sm text-zinc-100 placeholder-zinc-600 focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>
        <div>
          <label className="block text-xs text-zinc-500 mb-1">Notes</label>
          <textarea
            value={notes}
            onChange={e => setNotes(e.target.value)}
            placeholder="Optional tips or details for this step…"
            rows={2}
            className="w-full bg-zinc-800 border border-zinc-700 rounded-xl px-3.5 py-2.5 text-sm text-zinc-100 placeholder-zinc-600 resize-none focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>
        <div>
          <label className="block text-xs text-zinc-500 mb-1">Torque spec</label>
          <input
            type="text"
            value={torqueSpec}
            onChange={e => setTorqueSpec(e.target.value)}
            placeholder="e.g. 26 ft-lb"
            className="w-full bg-zinc-800 border border-zinc-700 rounded-xl px-3.5 py-2.5 text-sm text-zinc-100 placeholder-zinc-600 focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>
        {err && <p className="text-xs text-red-400">{err}</p>}
        <ModalActions onClose={onClose} onSubmit={submit} saving={saving} label="Add step" />
      </div>
    </DarkModal>
  )
}

// ── Add Part Modal ────────────────────────────────────────────────────────────

function AddPartModal({
  repairId,
  onClose,
  onSaved,
}: {
  repairId: string
  onClose: () => void
  onSaved: () => Promise<void>
}) {
  const api = useApi()
  const [name,       setName]       = useState('')
  const [partNumber, setPartNumber] = useState('')
  const [supplier,   setSupplier]   = useState('')
  const [cost,       setCost]       = useState('')
  const [saving,     setSaving]     = useState(false)
  const [err,        setErr]        = useState('')

  async function submit() {
    if (!name.trim()) { setErr('A name is required.'); return }
    if (cost && isNaN(parseFloat(cost))) { setErr('Cost must be a number.'); return }
    setSaving(true)
    setErr('')
    try {
      await api.repairs.createPart(repairId, {
        name:        name.trim(),
        part_number: partNumber.trim() || undefined,
        supplier:    supplier.trim() || undefined,
        cost:        cost.trim() || undefined,
      })
      await onSaved()
    } catch (e: any) {
      setErr(e.message ?? 'Something went wrong.')
      setSaving(false)
    }
  }

  return (
    <DarkModal title="Add part" onClose={onClose}>
      <div className="space-y-3">
        <div>
          <label className="block text-xs text-zinc-500 mb-1">
            Name <span className="text-red-400">*</span>
          </label>
          <input
            type="text"
            value={name}
            onChange={e => setName(e.target.value)}
            placeholder="e.g. Brake pad set"
            autoFocus
            className="w-full bg-zinc-800 border border-zinc-700 rounded-xl px-3.5 py-2.5 text-sm text-zinc-100 placeholder-zinc-600 focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>
        <div className="grid grid-cols-2 gap-2">
          <div>
            <label className="block text-xs text-zinc-500 mb-1">Part number</label>
            <input
              type="text"
              value={partNumber}
              onChange={e => setPartNumber(e.target.value)}
              placeholder="D1060-8J090"
              className="w-full bg-zinc-800 border border-zinc-700 rounded-xl px-3.5 py-2.5 text-sm text-zinc-100 placeholder-zinc-600 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
          <div>
            <label className="block text-xs text-zinc-500 mb-1">Supplier</label>
            <input
              type="text"
              value={supplier}
              onChange={e => setSupplier(e.target.value)}
              placeholder="RockAuto"
              className="w-full bg-zinc-800 border border-zinc-700 rounded-xl px-3.5 py-2.5 text-sm text-zinc-100 placeholder-zinc-600 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
        </div>
        <div>
          <label className="block text-xs text-zinc-500 mb-1">Cost ($)</label>
          <input
            type="text"
            value={cost}
            onChange={e => setCost(e.target.value)}
            placeholder="34.99"
            className="w-full bg-zinc-800 border border-zinc-700 rounded-xl px-3.5 py-2.5 text-sm text-zinc-100 placeholder-zinc-600 focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>
        {err && <p className="text-xs text-red-400">{err}</p>}
        <ModalActions onClose={onClose} onSubmit={submit} saving={saving} label="Add part" />
      </div>
    </DarkModal>
  )
}

// ── Shared modal primitives ───────────────────────────────────────────────────

function DarkModal({ title, onClose, children }: { title: string; onClose: () => void; children: React.ReactNode }) {
  return (
    <div className="fixed inset-0 z-30 flex items-end sm:items-center justify-center">
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={onClose} />
      <div className="relative w-full max-w-lg bg-zinc-900 border border-zinc-800 rounded-t-2xl sm:rounded-2xl shadow-2xl mx-0 sm:mx-4 p-5">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-base font-semibold text-zinc-100">{title}</h2>
          <button onClick={onClose} className="text-zinc-600 hover:text-zinc-300 transition-colors">
            <svg className="w-5 h-5" fill="none" stroke="currentColor" strokeWidth={2} viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
        {children}
      </div>
    </div>
  )
}

function ModalActions({
  onClose, onSubmit, saving, label,
}: { onClose: () => void; onSubmit: () => void; saving: boolean; label: string }) {
  return (
    <div className="flex gap-2 pt-1">
      <button
        onClick={onClose}
        className="flex-1 border border-zinc-700 text-zinc-400 hover:bg-zinc-800 font-medium text-sm py-2.5 rounded-xl transition-colors"
      >
        Cancel
      </button>
      <button
        onClick={onSubmit}
        disabled={saving}
        className="flex-1 bg-zinc-100 hover:bg-white disabled:opacity-40 text-zinc-900 font-medium text-sm py-2.5 rounded-xl transition-colors"
      >
        {saving ? 'Saving…' : label}
      </button>
    </div>
  )
}

function ConfirmDeleteModal({
  onClose, onConfirm, confirming,
}: { onClose: () => void; onConfirm: () => void; confirming: boolean }) {
  return (
    <DarkModal title="Delete repair?" onClose={onClose}>
      <p className="text-sm text-zinc-500 mb-6">
        This repair will be recoverable for 30 days. This action cannot be undone from the app.
      </p>
      <div className="flex gap-2">
        <button
          onClick={onClose}
          className="flex-1 border border-zinc-700 text-zinc-400 hover:bg-zinc-800 font-medium text-sm py-2.5 rounded-xl transition-colors"
        >
          Cancel
        </button>
        <button
          onClick={onConfirm}
          disabled={confirming}
          className="flex-1 bg-red-700 hover:bg-red-600 disabled:opacity-50 text-white font-medium text-sm py-2.5 rounded-xl transition-colors"
        >
          {confirming ? 'Deleting…' : 'Delete repair'}
        </button>
      </div>
    </DarkModal>
  )
}

function Section({
  title, count, accent, action, children,
}: {
  title: string
  count?: string
  accent?: boolean
  action?: React.ReactNode
  children: React.ReactNode
}) {
  return (
    <div className="bg-zinc-900 rounded-xl border border-zinc-800 overflow-hidden">
      <div className="px-4 py-3 border-b border-zinc-800 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="text-sm font-semibold text-zinc-100">{title}</span>
          {count !== undefined && (
            <span className={`text-xs px-2 py-0.5 rounded-full font-medium
              ${accent ? 'bg-green-950 text-green-400' : 'bg-zinc-800 text-zinc-500'}`}>
              {count}
            </span>
          )}
        </div>
        {action}
      </div>
      {children}
    </div>
  )
}

function NoteModal({
  body, onChange, onSubmit, onClose, submitting,
}: {
  body: string
  onChange: (v: string) => void
  onSubmit: () => void
  onClose: () => void
  submitting: boolean
}) {
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  useEffect(() => { textareaRef.current?.focus() }, [])

  return (
    <DarkModal title="Add note" onClose={onClose}>
      <textarea
        ref={textareaRef}
        value={body}
        onChange={e => onChange(e.target.value)}
        placeholder="What did you notice? Any observations, issues, or measurements…"
        rows={5}
        className="w-full bg-zinc-800 border border-zinc-700 rounded-xl px-3.5 py-3 text-sm text-zinc-100 placeholder-zinc-600 resize-none focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
        onKeyDown={e => { if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) onSubmit() }}
      />
      <p className="text-xs text-zinc-600 mt-2 mb-4">⌘ + Enter to save</p>
      <ModalActions onClose={onClose} onSubmit={onSubmit} saving={submitting} label="Save note" />
    </DarkModal>
  )
}

function LoadingState() {
  return (
    <div className="min-h-screen bg-black flex items-center justify-center">
      <div className="text-sm text-zinc-600">Loading repair…</div>
    </div>
  )
}

function ErrorState({ message }: { message: string }) {
  return (
    <div className="min-h-screen bg-black flex items-center justify-center">
      <div className="text-sm text-red-500">{message}</div>
    </div>
  )
}

function Chevron() {
  return (
    <svg className="w-3 h-3 text-zinc-700 flex-shrink-0" fill="none" stroke="currentColor" strokeWidth={2} viewBox="0 0 24 24">
      <path strokeLinecap="round" strokeLinejoin="round" d="M8.25 4.5l7.5 7.5-7.5 7.5" />
    </svg>
  )
}

function CheckIcon() {
  return (
    <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" strokeWidth={2.5} viewBox="0 0 24 24">
      <path strokeLinecap="round" strokeLinejoin="round" d="M4.5 12.75l6 6 9-13.5" />
    </svg>
  )
}