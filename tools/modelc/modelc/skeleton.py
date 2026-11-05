"""Skeletons described as data: bones with a parent and a rest position.

Every bone's rest orientation is the identity (aligned with the authoring axes), so a pose is
simply a rotation per bone around its head, in its parent's frame — easy to write by hand.
"""
import numpy as np


class Bone:
    def __init__(self, name, parent, head, tail):
        self.name, self.parent = name, parent
        self.head = np.array(head, dtype=float)
        self.tail = np.array(tail, dtype=float)


class Skeleton:
    def __init__(self, bones, mirror=True):
        """bones: list of (name, parent, head, tail). Names ending in .L get a .R twin (x flipped)."""
        out = []
        for name, parent, head, tail in bones:
            out.append(Bone(name, parent, head, tail))
        if mirror:
            for name, parent, head, tail in bones:
                if name.endswith(".L"):
                    rp = parent[:-2] + ".R" if parent and parent.endswith(".L") else parent
                    flip = lambda p: (-p[0], p[1], p[2])
                    out.append(Bone(name[:-2] + ".R", rp, flip(head), flip(tail)))
        self.bones = out
        self.index = {b.name: i for i, b in enumerate(out)}
        for b in out:
            if b.parent and b.parent not in self.index:
                raise ValueError(f"bone {b.name}: unknown parent {b.parent}")
        self.order = self._topo()

    def _topo(self):
        done, order = set(), []
        while len(order) < len(self.bones):
            for b in self.bones:
                if b.name not in done and (b.parent is None or b.parent in done):
                    done.add(b.name); order.append(b)
        return order

    def __getitem__(self, name):
        return self.bones[self.index[name]]

    def length(self, name):
        b = self[name]
        return float(np.linalg.norm(b.tail - b.head))

    def world(self, pose):
        """pose: {bone: (R 3x3, offset 3)} local rotation around the head + translation.
        Returns {bone: 4x4 world matrix}; the skinning matrix is world @ inverse(rest)."""
        W = {}
        for b in self.order:
            R, off = pose.get(b.name, (np.eye(3), np.zeros(3)))
            local = np.eye(4)
            local[:3, :3] = R
            if b.parent is None:
                local[:3, 3] = b.head + off
                W[b.name] = local
            else:
                p = self[b.parent]
                local[:3, 3] = (b.head - p.head) + off
                W[b.name] = W[b.parent] @ local
        return W

    def skin_matrices(self, pose):
        W = self.world(pose)
        out = np.zeros((len(self.bones), 4, 4))
        for b in self.bones:
            inv_rest = np.eye(4); inv_rest[:3, 3] = -b.head
            out[self.index[b.name]] = W[b.name] @ inv_rest
        return out
