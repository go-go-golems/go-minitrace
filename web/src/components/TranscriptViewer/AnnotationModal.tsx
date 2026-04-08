import Dialog from "@mui/material/Dialog";
import DialogTitle from "@mui/material/DialogTitle";
import DialogContent from "@mui/material/DialogContent";
import DialogActions from "@mui/material/DialogActions";
import Button from "@mui/material/Button";
import { AnnotationComposer, type AnnotationDraftTarget } from "./AnnotationComposer";

interface AnnotationModalProps {
  sessionId: string;
  target: AnnotationDraftTarget | null;
  onClose: () => void;
}

export function AnnotationModal({ sessionId, target, onClose }: AnnotationModalProps) {
  if (!target) return null;

  return (
    <Dialog
      open={!!target}
      onClose={onClose}
      maxWidth="sm"
      fullWidth
      PaperProps={{
        sx: {
          borderRadius: 2,
        },
      }}
    >
      <DialogTitle sx={{ pb: 1 }}>
        Add {target.scopeType} annotation
      </DialogTitle>
      <DialogContent>
        <AnnotationComposer
          sessionId={sessionId}
          target={target}
          compact={false}
          onCancel={onClose}
          onCreated={onClose}
        />
      </DialogContent>
      <DialogActions sx={{ px: 2, pb: 1.5 }}>
        <Button onClick={onClose} size="small">
          Cancel
        </Button>
      </DialogActions>
    </Dialog>
  );
}
